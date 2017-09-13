// +build !nofuse

// package fuse/mfs implements a fuse filesystem that interfaces
// with mfs
package mfsfuse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	core "github.com/ipfs/go-ipfs/core"
	dag "github.com/ipfs/go-ipfs/merkledag"
	mfs "github.com/ipfs/go-ipfs/mfs"
	path "github.com/ipfs/go-ipfs/path"
	ft "github.com/ipfs/go-ipfs/unixfs"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	fuse "gx/ipfs/QmaFNtBAXX4nVMQWbUqNysXyhevUj1k4B1y5uS45LC7Vw9/fuse"
	fs "gx/ipfs/QmaFNtBAXX4nVMQWbUqNysXyhevUj1k4B1y5uS45LC7Vw9/fuse/fs"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

var log = logging.Logger("fuse/mfs")

// FileSystem is the readwrite IPNS Fuse Filesystem.
type FileSystem struct {
	mfsr *mfs.Root
}

// NewFileSystem constructs new fs using given core.IpfsNode instance.
func NewFileSystem(mfsr *mfs.Root) *FileSystem {
	return &FileSystem{mfsr: mfsr}
}

// Root constructs the Root of the filesystem, a Root object.
func (f *FileSystem) Root() (fs.Node, error) {
	switch nd := f.mfsr.GetValue().(type) {
	case *mfs.Directory:
		return &Directory{nd}, nil
	case *mfs.File:
		return nil, fmt.Errorf("root cannot be a file")
	default:
		err := fmt.Errorf("unrecognized node type: %#v", nd)
		log.Error("get root: ", err)
		return nil, err
	}
}

func (f *FileSystem) Destroy() {
	if err := f.mfsr.Close(); err != nil {
		log.Error("calling close on mfs root: ", err)
	}
}

func ipnsPubFunc(ipfs *core.IpfsNode, k ci.PrivKey) mfs.PubFunc {
	return func(ctx context.Context, c *cid.Cid) error {
		return ipfs.Namesys.Publish(ctx, k, path.FromCid(c))
	}
}

// Directory is wrapper over an mfs directory to satisfy the fuse fs interface
type Directory struct {
	dir *mfs.Directory
}

type FileNode struct {
	fi *mfs.File
}

// File is wrapper over an mfs file to satisfy the fuse fs interface
type File struct {
	fi mfs.FileDescriptor
}

// Attr returns the attributes of a given node.
func (d *Directory) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("Directory Attr")
	a.Mode = os.ModeDir | 0555
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

// Attr returns the attributes of a given node.
func (fi *FileNode) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("File Attr")
	size, err := fi.fi.Size()
	if err != nil {
		// In this case, the dag node in question may not be unixfs
		return fmt.Errorf("fuse/ipns: failed to get file.Size(): %s", err)
	}
	a.Mode = os.FileMode(0666)
	a.Size = uint64(size)
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

// Lookup performs a lookup under this node.
func (s *Directory) Lookup(ctx context.Context, name string) (fs.Node, error) {
	child, err := s.dir.Child(name)
	if err != nil {
		// todo: make this error more versatile.
		return nil, fuse.ENOENT
	}

	switch child := child.(type) {
	case *mfs.Directory:
		return &Directory{dir: child}, nil
	case *mfs.File:
		return &FileNode{fi: child}, nil
	default:
		// NB: if this happens, we do not want to continue, unpredictable behaviour
		// may occur.
		panic("invalid type found under directory. programmer error.")
	}
}

// ReadDirAll reads the link structure as directory entries
func (dir *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	listing, err := dir.dir.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, entry := range listing {
		dirent := fuse.Dirent{Name: entry.Name}

		switch mfs.NodeType(entry.Type) {
		case mfs.TDir:
			dirent.Type = fuse.DT_Dir
		case mfs.TFile:
			dirent.Type = fuse.DT_File
		}

		entries = append(entries, dirent)
	}

	if len(entries) > 0 {
		return entries, nil
	}
	return nil, fuse.ENOENT
}

func (fi *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	_, err := fi.fi.Seek(req.Offset, io.SeekStart)
	if err != nil {
		return err
	}

	fisize, err := fi.fi.Size()
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	readsize := min(req.Size, int(fisize-req.Offset))
	n, err := fi.fi.CtxReadFull(ctx, resp.Data[:readsize])
	resp.Data = resp.Data[:n]
	return err
}

func (fi *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	// TODO: at some point, ensure that WriteAt here respects the context
	wrote, err := fi.fi.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Size = wrote
	return nil
}

func (fi *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	errs := make(chan error, 1)
	go func() {
		errs <- fi.fi.Flush()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (fi *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.Size() {
		cursize, err := fi.fi.Size()
		if err != nil {
			return err
		}
		if cursize != int64(req.Size) {
			err := fi.fi.Truncate(int64(req.Size))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Fsync flushes the content in the file to disk, but does not
// update the dag tree internally
func (fi *FileNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	errs := make(chan error, 1)
	go func() {
		errs <- fi.fi.Sync()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (fi *File) Forget() {
	err := fi.fi.Sync()
	if err != nil {
		log.Debug("forget file error: ", err)
	}
}

func (dir *Directory) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	child, err := dir.dir.Mkdir(req.Name)
	if err != nil {
		return nil, err
	}

	return &Directory{dir: child}, nil
}

func (fi *FileNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	var mfsflag int
	switch {
	case req.Flags.IsReadOnly():
		mfsflag = mfs.OpenReadOnly
	case req.Flags.IsWriteOnly():
		mfsflag = mfs.OpenWriteOnly
	case req.Flags.IsReadWrite():
		mfsflag = mfs.OpenReadWrite
	default:
		return nil, errors.New("unsupported flag type")
	}

	fd, err := fi.fi.Open(mfsflag, true)
	if err != nil {
		return nil, err
	}

	if req.Flags&fuse.OpenTruncate != 0 {
		if req.Flags.IsReadOnly() {
			log.Error("tried to open a readonly file with truncate")
			return nil, fuse.ENOTSUP
		}
		log.Info("Need to truncate file!")
		err := fd.Truncate(0)
		if err != nil {
			return nil, err
		}
	} else if req.Flags&fuse.OpenAppend != 0 {
		log.Info("Need to append to file!")
		if req.Flags.IsReadOnly() {
			log.Error("tried to open a readonly file with append")
			return nil, fuse.ENOTSUP
		}

		_, err := fd.Seek(0, io.SeekEnd)
		if err != nil {
			log.Error("seek reset failed: ", err)
			return nil, err
		}
	}

	return &File{fi: fd}, nil
}

func (fi *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return fi.fi.Close()
}

func (dir *Directory) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	// New 'empty' file
	nd := dag.NodeWithData(ft.FilePBData(nil, 0))
	err := dir.dir.AddChild(req.Name, nd)
	if err != nil {
		return nil, nil, err
	}

	child, err := dir.dir.Child(req.Name)
	if err != nil {
		return nil, nil, err
	}

	fi, ok := child.(*mfs.File)
	if !ok {
		return nil, nil, errors.New("child creation failed")
	}

	nodechild := &FileNode{fi: fi}

	var openflag int
	switch {
	case req.Flags.IsReadOnly():
		openflag = mfs.OpenReadOnly
	case req.Flags.IsWriteOnly():
		openflag = mfs.OpenWriteOnly
	case req.Flags.IsReadWrite():
		openflag = mfs.OpenReadWrite
	default:
		return nil, nil, errors.New("unsupported open mode")
	}

	fd, err := fi.Open(openflag, true)
	if err != nil {
		return nil, nil, err
	}

	return nodechild, &File{fi: fd}, nil
}

func (dir *Directory) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	err := dir.dir.Unlink(req.Name)
	if err != nil {
		return fuse.ENOENT
	}
	return nil
}

// Rename implements NodeRenamer
func (dir *Directory) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	cur, err := dir.dir.Child(req.OldName)
	if err != nil {
		return err
	}

	err = dir.dir.Unlink(req.OldName)
	if err != nil {
		return err
	}

	switch newDir := newDir.(type) {
	case *Directory:
		nd, err := cur.GetNode()
		if err != nil {
			return err
		}

		err = newDir.dir.AddChild(req.NewName, nd)
		if err != nil {
			return err
		}
	case *FileNode:
		log.Error("Cannot move node into a file!")
		return fuse.EPERM
	default:
		log.Error("Unknown node type for rename target dir!")
		return errors.New("Unknown fs node type!")
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type ipnsDirectory interface {
	fs.HandleReadDirAller
	fs.Node
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeStringLookuper
}

var _ ipnsDirectory = (*Directory)(nil)

type ipnsFile interface {
	fs.HandleFlusher
	fs.HandleReader
	fs.HandleWriter
	fs.HandleReleaser
}

type ipnsFileNode interface {
	fs.Node
	fs.NodeFsyncer
	fs.NodeOpener
}

var _ ipnsFileNode = (*FileNode)(nil)
var _ ipnsFile = (*File)(nil)
