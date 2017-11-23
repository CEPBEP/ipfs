package coreapi

import (
	"context"
	"io"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	coreunix "github.com/ipfs/go-ipfs/core/coreunix"
	ufs "github.com/ipfs/go-ipfs/unixfs"
	uio "github.com/ipfs/go-ipfs/unixfs/io"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	ipld "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
)

type UnixfsAPI CoreAPI

func (api *UnixfsAPI) Add(ctx context.Context, r io.Reader) (coreiface.Path, error) {
	k, err := coreunix.AddWithContext(ctx, api.node, r)
	if err != nil {
		return nil, err
	}
	c, err := cid.Decode(k)
	if err != nil {
		return nil, err
	}
	return ParseCid(c), nil
}

func (api *UnixfsAPI) Cat(ctx context.Context, p coreiface.Path) (coreiface.Path, coreiface.Reader, error) {
	rp, dagnode, err := api.core().ResolveNode(ctx, p)
	if err != nil {
		return nil, nil, err
	}

	r, err := uio.NewDagReader(ctx, dagnode, api.node.DAG)
	if err == uio.ErrIsDir {
		return nil, nil, coreiface.ErrIsDir
	} else if err != nil {
		return nil, nil, err
	}
	return rp, r, nil
}

func (api *UnixfsAPI) Ls(ctx context.Context, p coreiface.Path) (coreiface.Path, []*coreiface.Link, error) {
	rp, dir, err := api.LsDir(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	links, err := dir.Links(ctx)
	if err != nil {
		return nil, nil, err
	}
	return rp, links, nil
}

func (api *UnixfsAPI) LsDir(ctx context.Context, p coreiface.Path) (coreiface.Path, coreiface.UnixfsDir, error) {
	rp, dagnode, err := api.core().ResolveNode(ctx, p)
	if err != nil {
		return nil, nil, err
	}

	dir, err := uio.NewDirectoryFromNode(api.node.DAG, dagnode)
	if err == uio.ErrNotADir {
		return rp, nil, coreiface.ErrNotADir
	}

	return rp, unixfsDir{dir}, nil
}

func (api *UnixfsAPI) core() coreiface.CoreAPI {
	return (*CoreAPI)(api)
}

// move to dageditor
func EmptyUnixfsDir() coreiface.Node {
	return ufs.EmptyDirNode()
}

type unixfsDir struct {
	*uio.Directory
}

func (d unixfsDir) Node() (coreiface.Node, error) {
	dagnode, err := d.Directory.GetNode()
	if err != nil {
		return nil, err
	}
	return (coreiface.Node)(dagnode), nil
}

func (d unixfsDir) Links(ctx context.Context) ([]*coreiface.Link, error) {
	var links []*coreiface.Link
	err := d.ForEachLink(ctx, func(link *coreiface.Link) error {
		links = append(links, link)
		return nil
	})
	return links, err
}

func (d unixfsDir) ForEachLink(ctx context.Context, f func(*coreiface.Link) error) error {
	return d.Directory.ForEachLink(ctx, func(link *ipld.Link) error {
		return f((*coreiface.Link)(link))
	})
}

func (d unixfsDir) Find(ctx context.Context, name string) (coreiface.Node, error) {
	return d.Directory.Find(ctx, name)
}
