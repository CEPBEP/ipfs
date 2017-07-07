package namesys

import (
	"context"
	"sync"
	"testing"
	"time"

	path "github.com/ipfs/go-ipfs/path"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	testutil "github.com/ipfs/go-ipfs/thirdparty/testutil"

	ci "gx/ipfs/QmP1DfoUjiWH2ZBo1PBH6FupdBucbDepx3HpWmEY6JMUpY/go-libp2p-crypto"
	routing "gx/ipfs/QmP1wMAqk6aZYRZirbaAwmrNeqFRgQrwBt3orUtvSa1UYD/go-libp2p-routing"
	floodsub "gx/ipfs/QmUpeULWfmtsgCnfuRN3BHsfhHvBxNphoYh4La4CMxGt2Z/floodsub"
	p2phost "gx/ipfs/QmUywuGNZoUKV8B9iyvup9bPkLiMrhTsyVMkeSXW5VxAfC/go-libp2p-host"
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	netutil "gx/ipfs/Qma2j8dYePrvN5DoNgwh1uAuu3FFtEtrUQFmr737ws8nCp/go-libp2p-netutil"
	bhost "gx/ipfs/Qma4Xhhqtr9tpV814eNjbLHzjuDaRjs96XLcZPJiR742ZV/go-libp2p-blankhost"
	peer "gx/ipfs/QmdS9KpbDyPrieswibZhkod1oXqRwZJrUPzxCofAMWpFGq/go-libp2p-peer"
)

func newNetHost(ctx context.Context, t *testing.T) p2phost.Host {
	netw := netutil.GenSwarmNetwork(t, ctx)
	return bhost.NewBlankHost(netw)
}

func newNetHosts(ctx context.Context, t *testing.T, n int) []p2phost.Host {
	var out []p2phost.Host

	for i := 0; i < n; i++ {
		h := newNetHost(ctx, t)
		out = append(out, h)
	}

	return out
}

// PubKeyFetcher implementation with a global key store
type mockKeyStore struct {
	keys map[peer.ID]ci.PubKey
	mx   sync.Mutex
}

func (m *mockKeyStore) addPubKey(id peer.ID, pkey ci.PubKey) {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.keys[id] = pkey
}

func (m *mockKeyStore) getPubKey(id peer.ID) (ci.PubKey, error) {
	m.mx.Lock()
	defer m.mx.Unlock()
	pkey, ok := m.keys[id]
	if ok {
		return pkey, nil
	}

	return nil, routing.ErrNotFound
}

func (m *mockKeyStore) GetPublicKey(ctx context.Context, id peer.ID) (ci.PubKey, error) {
	return m.getPubKey(id)
}

func newMockKeyStore() *mockKeyStore {
	return &mockKeyStore{
		keys: make(map[peer.ID]ci.PubKey),
	}
}

// ConentRouting mock
func newMockRouting(ms mockrouting.Server, ks *mockKeyStore, host p2phost.Host) routing.ContentRouting {
	id := host.ID()

	privk := host.Peerstore().PrivKey(id)
	pubk := host.Peerstore().PubKey(id)
	pi := host.Peerstore().PeerInfo(id)

	ks.addPubKey(id, pubk)
	return ms.Client(testutil.NewIdentity(id, pi.Addrs[0], privk, pubk))
}

func newMockRoutingForHosts(ms mockrouting.Server, ks *mockKeyStore, hosts []p2phost.Host) []routing.ContentRouting {
	rs := make([]routing.ContentRouting, len(hosts))
	for i := 0; i < len(hosts); i++ {
		rs[i] = newMockRouting(ms, ks, hosts[i])
	}
	return rs
}

// tests
func TestPubsubPublishResolve(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ms := mockrouting.NewServer()
	ks := newMockKeyStore()

	pubhost := newNetHost(ctx, t)
	pubmr := newMockRouting(ms, ks, pubhost)
	pub := NewPubsubPublisher(ctx, ds.NewMapDatastore(), pubhost, pubmr, floodsub.NewFloodSub(ctx, pubhost))
	privk := pubhost.Peerstore().PrivKey(pubhost.ID())

	name := "/ipns/" + pubhost.ID().Pretty()

	reshosts := newNetHosts(ctx, t, 20)
	resmrs := newMockRoutingForHosts(ms, ks, reshosts)
	res := make([]Resolver, len(reshosts))
	for i := 0; i < len(res); i++ {
		res[i] = NewPubsubResolver(ctx, reshosts[i], resmrs[i], ks, floodsub.NewFloodSub(ctx, reshosts[i]))
	}

	time.Sleep(time.Millisecond * 100)
	for i := 0; i < len(res); i++ {
		checkResolveNotFound(ctx, t, res[i], name)
	}

	// let the bootstrap finish
	time.Sleep(time.Second * 1)

	val := path.Path("/ipfs/QmP1DfoUjiWH2ZBo1PBH6FupdBucbDepx3HpWmEY6JMUpY")
	err := pub.Publish(ctx, privk, val)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
	for i := 0; i < len(res); i++ {
		checkResolve(ctx, t, res[i], name, val)
	}

	val = path.Path("/ipfs/QmP1wMAqk6aZYRZirbaAwmrNeqFRgQrwBt3orUtvSa1UYD")
	err = pub.Publish(ctx, privk, val)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
	for i := 0; i < len(res); i++ {
		checkResolve(ctx, t, res[i], name, val)
	}
}

func checkResolveNotFound(ctx context.Context, t *testing.T, resolver Resolver, name string) {
	_, err := resolver.Resolve(ctx, name)
	if err != ds.ErrNotFound {
		t.Fatalf("unexpected value: %#v", err)
	}
}

func checkResolve(ctx context.Context, t *testing.T, resolver Resolver, name string, val path.Path) {
	xval, err := resolver.Resolve(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if xval != val {
		t.Fatalf("resolver resolves to unexpected value %s %s", val, xval)
	}
}
