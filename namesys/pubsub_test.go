package namesys

import (
	"context"
	"sync"
	"testing"

	routing "gx/ipfs/QmP1wMAqk6aZYRZirbaAwmrNeqFRgQrwBt3orUtvSa1UYD/go-libp2p-routing"
	p2phost "gx/ipfs/QmUywuGNZoUKV8B9iyvup9bPkLiMrhTsyVMkeSXW5VxAfC/go-libp2p-host"
	pstore "gx/ipfs/QmXZSd1qR5BxZkPyuwfT5jpqQFScZccoZvDneXsKzCNHWX/go-libp2p-peerstore"
	netutil "gx/ipfs/Qma2j8dYePrvN5DoNgwh1uAuu3FFtEtrUQFmr737ws8nCp/go-libp2p-netutil"
	cid "gx/ipfs/Qma4RJSuh7mMeJQYCqMbKzekn6EwBo7HEs5AQYjVRMQATB/go-cid"
	bhost "gx/ipfs/Qma4Xhhqtr9tpV814eNjbLHzjuDaRjs96XLcZPJiR742ZV/go-libp2p-blankhost"
	peer "gx/ipfs/QmdS9KpbDyPrieswibZhkod1oXqRwZJrUPzxCofAMWpFGq/go-libp2p-peer"
)

func newNetHosts(t *testing.T, ctx context.Context, n int) []p2phost.Host {
	var out []p2phost.Host

	for i := 0; i < n; i++ {
		netw := netutil.GenSwarmNetwork(t, ctx)
		h := bhost.NewBlankHost(netw)
		out = append(out, h)
	}

	return out
}

type mockDHT struct {
	prov map[peer.ID][]pstore.PeerInfo
}

func newDHT() *mockDHT {

}
