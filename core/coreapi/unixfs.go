package coreapi

import (
	"context"
	"errors"
	"io"

	core "github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	coreunix "github.com/ipfs/go-ipfs/core/coreunix"
	mdag "github.com/ipfs/go-ipfs/merkledag"
	uio "github.com/ipfs/go-ipfs/unixfs/io"
	ftpb "github.com/ipfs/go-ipfs/unixfs/pb"

	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	cid "gx/ipfs/QmcTcsTvfaeEBRFo1TkFgT8sRmgi1n1LTZpecfVP8fzpGD/go-cid"
)

type UnixfsAPI struct {
	node *core.IpfsNode
}

func NewUnixfsAPI(n *core.IpfsNode) coreiface.UnixfsAPI {
	api := &UnixfsAPI{n}
	return api
}

func (api *UnixfsAPI) Add(ctx context.Context, r io.Reader) (*cid.Cid, error) {
	k, err := coreunix.AddWithContext(ctx, api.node, r)
	if err != nil {
		return nil, err
	}
	return cid.Decode(k)
}

func (api *UnixfsAPI) Cat(ctx context.Context, p string) (coreiface.Reader, error) {
	dagnode, err := resolve(ctx, api.node, p)
	if err != nil {
		return nil, err
	}

	r, err := uio.NewDagReader(ctx, dagnode, api.node.DAG)
	switch err {
	case uio.ErrIsDir:
		return nil, coreiface.ErrIsDir
	case uio.ErrCantReadSymlinks:
		return nil, coreiface.ErrIsSymLink
	default:
	}
	return r, err;
}

func (api *UnixfsAPI) Ls(ctx context.Context, p string) ([]*coreiface.Link, error) {
	dagnode, err := resolve(ctx, api.node, p)
	if err != nil {
		return nil, err
	}

	l := dagnode.Links()
	links := make([]*coreiface.Link, len(l))
	for i, l := range l {
		links[i] = &coreiface.Link{l.Name, l.Size, l.Cid}
	}
	return links, nil
}

var NotASymLink = errors.New("not a symbolic link")

func (api *UnixfsAPI) ReadSymLink(ctx context.Context, p string) (string, error) {
	dagnode, err := resolve(ctx, api.node, p)
	if err != nil {
		return "", err
	}
	switch n := dagnode.(type) {
	case *mdag.ProtoNode:
		pb := new(ftpb.Data)
		if err := proto.Unmarshal(n.Data(), pb); err != nil {
			return "", err
		}
		switch pb.GetType() {
		case ftpb.Data_Symlink:
			return string(pb.GetData()), nil
		default:
			return "", NotASymLink
		}
	default:
		return "", NotASymLink
	}
}
