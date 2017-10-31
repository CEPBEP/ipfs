// Package iface provides the interfaces of the IPFS Core API.
//
// TODO: package should be named coreapi instead.
package iface

import (
	"context"
	"errors"
	"io"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	ipld "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
)

type Path interface {
	String() string
	Cid() *cid.Cid
	Root() *cid.Cid
	Resolved() bool
}

// TODO: should we really copy these?
//       if we didn't, godoc would generate nice links straight to go-ipld-format
type Node ipld.Node
type Link ipld.Link

type CoreAPI interface {
	Unixfs() UnixfsAPI
	Object() ObjectAPI
	ResolvePath(context.Context, Path) (Path, error)
	ResolveNode(context.Context, Path) (Node, error)
}

type UnixfsAPI interface {
	Add(context.Context, io.Reader) (Path, error)
	Cat(context.Context, Path) (Reader, error)
	Ls(context.Context, Path) ([]*Link, error)
}

type Reader interface {
	io.ReadSeeker
	io.Closer
}

type ObjectAPI interface {
	Get(context.Context, *cid.Cid) (*Object, error)
	Put(context.Context, Object) (*cid.Cid, error)
	AddLink(ctx context.Context, root *cid.Cid, path string, target *cid.Cid) (*cid.Cid, error)
	RmLink(ctx context.Context, root *cid.Cid, path string) (*cid.Cid, error)
	// New() (cid.Cid, Object)
	// Links(string) ([]*Link, error)
	// Data(string) (Reader, error)
	// Stat(string) (ObjectStat, error)
	// SetData(string, Reader) (cid.Cid, error)
	// AppendData(string, Data) (cid.Cid, error)
}

// type ObjectStat struct {
// 	Cid            cid.Cid
// 	NumLinks       int
// 	BlockSize      int
// 	LinksSize      int
// 	DataSize       int
// 	CumulativeSize int
// }

// ErrIsDir is returned by Cat() if the Path which was passed resolves to a unixfs directory.
var ErrIsDir = errors.New("object is a directory")

// ErrOffline is returned if the IPFS node backing this Core API instance was started with --offline.
var ErrOffline = errors.New("can't resolve, ipfs node is offline")
