// Package namecache implements background following (resolution and pinning) of names
package namecache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	bstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	merkledag "github.com/ipfs/go-ipfs/merkledag"
	namesys "github.com/ipfs/go-ipfs/namesys"
	path "github.com/ipfs/go-ipfs/path"
	pin "github.com/ipfs/go-ipfs/pin"
	uio "github.com/ipfs/go-ipfs/unixfs/io"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

const (
	followInterval = 60 * time.Minute
	resolveTimeout = 60 * time.Second
)

var log = logging.Logger("namecache")

// NameCache represents a following cache of names
type NameCache interface {
	// Follow starts following name, pinning it if pinit is true
	Follow(name string, pinit bool) error
	// Unofollow cancels a follow
	Unfollow(name string) error
	// ListFollows returns a list of followed names
	ListFollows() []string
}

type nameCache struct {
	nsys    namesys.NameSystem
	pinning pin.Pinner
	dag     merkledag.DAGService
	bstore  bstore.GCBlockstore

	ctx     context.Context
	follows map[string]func()
	mx      sync.Mutex
}

func NewNameCache(ctx context.Context, nsys namesys.NameSystem, pinning pin.Pinner, dag merkledag.DAGService) NameCache {
	return &nameCache{
		ctx:     ctx,
		nsys:    nsys,
		pinning: pinning,
		dag:     dag,
		follows: make(map[string]func()),
	}
}

// Follow spawns a goroutine that periodically resolves a name
// and (when pinit is true) pins it in the background
func (nc *nameCache) Follow(name string, pinit bool) error {
	nc.mx.Lock()
	defer nc.mx.Unlock()

	if _, ok := nc.follows[name]; ok {
		return fmt.Errorf("Already following %s", name)
	}

	ctx, cancel := context.WithCancel(nc.ctx)
	go nc.followName(ctx, name, pinit)
	nc.follows[name] = cancel

	return nil
}

// Unfollow cancels a follow
func (nc *nameCache) Unfollow(name string) error {
	nc.mx.Lock()
	defer nc.mx.Unlock()

	cancel, ok := nc.follows[name]
	if ok {
		cancel()
		delete(nc.follows, name)
		return nil
	}

	return fmt.Errorf("Unknown name %s", name)
}

// ListFollows returns a list of names currently being followed
func (nc *nameCache) ListFollows() []string {
	nc.mx.Lock()
	defer nc.mx.Unlock()

	follows := make([]string, 0)
	for name, _ := range nc.follows {
		follows = append(follows, name)
	}

	return follows
}

func (nc *nameCache) followName(ctx context.Context, name string, pinit bool) {
	// if cid != nil, we have created a new pin that is updated on changes and
	// unpinned on cancel
	cid, err := nc.resolveAndPin(ctx, name, pinit)
	if err != nil {
		log.Errorf("Error following %s: %s", name, err.Error())
	}

	ticker := time.NewTicker(followInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if cid != nil {
				cid, err = nc.resolveAndUpdate(ctx, name, cid)
			} else {
				cid, err = nc.resolveAndPin(ctx, name, pinit)
			}

			if err != nil {
				log.Errorf("Error following %s: %s", name, err.Error())
			}

		case <-ctx.Done():
			if cid != nil {
				err = nc.unpin(cid)
				if err != nil {
					log.Errorf("Error unpinning followed %s: %s", name, err.Error())
				}
			}
			return
		}
	}
}

func (nc *nameCache) resolveAndPin(ctx context.Context, name string, pinit bool) (*cid.Cid, error) {
	ptr, err := nc.resolve(ctx, name)
	if err != nil {
		return nil, err
	}

	if !pinit {
		return nil, nil
	}

	cid, err := pathToCid(ptr)
	if err != nil {
		return nil, err
	}

	defer nc.bstore.PinLock().Unlock()

	_, pinned, err := nc.pinning.IsPinned(cid)
	if pinned || err != nil {
		return nil, err
	}

	n, err := nc.pathToNode(ctx, ptr)
	if err != nil {
		return nil, err
	}

	log.Debugf("pinning %s", cid.String())

	err = nc.pinning.Pin(ctx, n, true)
	if err != nil {
		return nil, err
	}

	err = nc.pinning.Flush()

	return cid, err
}

func (nc *nameCache) resolveAndUpdate(ctx context.Context, name string, cid *cid.Cid) (*cid.Cid, error) {

	ptr, err := nc.resolve(ctx, name)
	if err != nil {
		return nil, err
	}

	ncid, err := pathToCid(ptr)
	if err != nil {
		return nil, err
	}

	if ncid.Equals(cid) {
		return cid, nil
	}

	defer nc.bstore.PinLock().Unlock()

	err = nc.pinning.Update(ctx, cid, ncid, true)
	if err != nil {
		return cid, err
	}

	err = nc.pinning.Flush()

	return ncid, err
}

func (nc *nameCache) unpin(cid *cid.Cid) error {
	defer nc.bstore.PinLock().Unlock()

	err := nc.pinning.Unpin(nc.ctx, cid, true)
	if err != nil {
		return err
	}

	return nc.pinning.Flush()
}

func (nc *nameCache) resolve(ctx context.Context, name string) (path.Path, error) {
	log.Debugf("resolving %s", name)

	if !strings.HasPrefix(name, "/ipns/") {
		name = "/ipns/" + name
	}

	rctx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()

	p, err := nc.nsys.Resolve(rctx, name)
	if err != nil {
		return "", err
	}

	log.Debugf("resolved %s to %s", name, p)

	return p, nil
}

func pathToCid(p path.Path) (*cid.Cid, error) {
	return cid.Decode(p.Segments()[1])
}

func (nc *nameCache) pathToNode(ctx context.Context, p path.Path) (node.Node, error) {
	r := &path.Resolver{
		DAG:         nc.dag,
		ResolveOnce: uio.ResolveUnixfsOnce,
	}

	return r.ResolvePath(ctx, p)
}
