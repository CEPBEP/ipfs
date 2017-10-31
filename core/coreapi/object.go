package coreapi

import (
	"context"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	cid "gx/ipfs/QmcEcrBAMrwMyhSjXt4yfyPpzgSuV8HLHavnfmiKCSRqZU/go-cid"
)

type ObjectAPI CoreAPI

func (api *ObjectAPI) Get(context.Context, string) (*coreiface.Object, error) {
	dagnode, err := resolve(ctx, api.node, p)
	if err != nil {
		return nil, err
	}

	obj := &coreiface.Object{
		Data:  dagnode.RawData(),
		Links: dagnode.Links(),
	}
	return obj, nil
}

func (api *ObjectAPI) Put(context.Context, coreiface.Object) (*cid.Cid, error) {
	return cid.Decode("Qmfoobar")
}
