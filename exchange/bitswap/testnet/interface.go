package bitswap

import (
	peer "gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	"gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"

	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"

	"github.com/ipfs/go-ipfs/providers"
)

type Network interface {
	Adapter(testutil.Identity) bsnet.BitSwapNetwork
	Providers() providers.Interface

	HasPeer(peer.ID) bool
}
