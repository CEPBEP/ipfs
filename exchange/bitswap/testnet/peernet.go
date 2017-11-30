package bitswap

import (
	"context"

	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	pr "github.com/ipfs/go-ipfs/providers"
	providers "github.com/ipfs/go-ipfs/providers"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"

	peer "gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	mockpeernet "gx/ipfs/Qma23bpHwQrQyvKeBemaeJh7sAoRHggPkgnge1B9489ff5/go-libp2p/p2p/net/mock"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
)

type peernet struct {
	mockpeernet.Mocknet
	routingserver mockrouting.Server
	providers     providers.Interface
}

func StreamNet(ctx context.Context, net mockpeernet.Mocknet, rs mockrouting.Server) (Network, error) {
	return &peernet{net, rs, nil}, nil
}

func (pn *peernet) Adapter(p testutil.Identity) bsnet.BitSwapNetwork {
	client, err := pn.Mocknet.AddPeer(p.PrivateKey(), p.Address())
	if err != nil {
		panic(err.Error())
	}
	routing := pn.routingserver.ClientWithDatastore(context.TODO(), p, ds.NewMapDatastore())
	pn.providers = pr.NewProviders(context.TODO(), routing, client)

	return bsnet.NewFromIpfsHost(client, routing)
}

func (pn *peernet) HasPeer(p peer.ID) bool {
	for _, member := range pn.Mocknet.Peers() {
		if p == member {
			return true
		}
	}
	return false
}

func (pn *peernet) Providers() providers.Interface {
	return pn.providers
}

var _ Network = (*peernet)(nil)
