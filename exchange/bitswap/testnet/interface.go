package bitswap

import (
	"gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	bsnet "github.com/ipfs/go-ipfs/exchange/bitswap/network"
	"github.com/ipfs/go-ipfs/providers"
)

type Network interface {
	Adapter(testutil.Identity) bsnet.BitSwapNetwork
	Providers() providers.Interface

	HasPeer(peer.ID) bool
}
