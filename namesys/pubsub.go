package namesys

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	pb "github.com/ipfs/go-ipfs/namesys/pb"
	path "github.com/ipfs/go-ipfs/path"
	dshelp "github.com/ipfs/go-ipfs/thirdparty/ds-help"

	ci "gx/ipfs/QmP1DfoUjiWH2ZBo1PBH6FupdBucbDepx3HpWmEY6JMUpY/go-libp2p-crypto"
	routing "gx/ipfs/QmP1wMAqk6aZYRZirbaAwmrNeqFRgQrwBt3orUtvSa1UYD/go-libp2p-routing"
	floodsub "gx/ipfs/QmUpeULWfmtsgCnfuRN3BHsfhHvBxNphoYh4La4CMxGt2Z/floodsub"
	p2phost "gx/ipfs/QmUywuGNZoUKV8B9iyvup9bPkLiMrhTsyVMkeSXW5VxAfC/go-libp2p-host"
	mh "gx/ipfs/QmVGtdTZdTFaLsaj2RwdVG8jcjNNcp1DE914DKZ2kHmXHw/go-multihash"
	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	record "gx/ipfs/QmWYCqr6UDqqD1bfRybaAPtbAqcN3TSJpveaBXMwbQ3ePZ/go-libp2p-record"
	dhtpb "gx/ipfs/QmWYCqr6UDqqD1bfRybaAPtbAqcN3TSJpveaBXMwbQ3ePZ/go-libp2p-record/pb"
	u "gx/ipfs/QmWbjfz3u6HkAdPh34dgPchGbQjob6LXLhAeCGii2TX69n/go-ipfs-util"
	pstore "gx/ipfs/QmXZSd1qR5BxZkPyuwfT5jpqQFScZccoZvDneXsKzCNHWX/go-libp2p-peerstore"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	cid "gx/ipfs/Qma4RJSuh7mMeJQYCqMbKzekn6EwBo7HEs5AQYjVRMQATB/go-cid"
	peer "gx/ipfs/QmdS9KpbDyPrieswibZhkod1oXqRwZJrUPzxCofAMWpFGq/go-libp2p-peer"
)

type pubsubPublisher struct {
	ctx  context.Context
	ds   ds.Datastore
	host p2phost.Host
	cr   routing.ContentRouting
	ps   *floodsub.PubSub
	subs map[string]struct{}
	mx   sync.Mutex
}

type pubsubResolver struct {
	ctx  context.Context
	ds   ds.Datastore
	host p2phost.Host
	cr   routing.ContentRouting
	pkf  routing.PubKeyFetcher
	ps   *floodsub.PubSub
	subs map[string]*floodsub.Subscription
	mx   sync.Mutex
}

// NewPubsubPublisher constructs a new Publisher that publishes IPNS records through pubsub.
// The constructor interface is complicated by the need to bootstrap the pubsub topic.
// This could be greatly simplified if the pubsub implementation handled bootstrap itself
func NewPubsubPublisher(ctx context.Context, ds ds.Datastore, host p2phost.Host, cr routing.ContentRouting, ps *floodsub.PubSub) Publisher {
	return &pubsubPublisher{
		ctx:  ctx,
		ds:   ds,
		host: host, // needed for pubsub bootstrap
		cr:   cr,   // needed for pubsub bootstrap
		ps:   ps,
		subs: make(map[string]struct{}),
	}
}

// NewPubsubResolver constructs a new Resolver that resolves IPNS records through pubsub.
// same as above for pubsub bootstrap dependencies
func NewPubsubResolver(ctx context.Context, host p2phost.Host, cr routing.ContentRouting, pkf routing.PubKeyFetcher, ps *floodsub.PubSub) Resolver {
	return &pubsubResolver{
		ctx:  ctx,
		ds:   ds.NewMapDatastore(),
		host: host, // needed for pubsub bootstrap
		cr:   cr,   // needed for pubsub bootstrap
		pkf:  pkf,
		ps:   ps,
		subs: make(map[string]*floodsub.Subscription),
	}
}

// PubsubPublisher implementation
func (p *pubsubPublisher) Publish(ctx context.Context, k ci.PrivKey, value path.Path) error {
	return p.PublishWithEOL(ctx, k, value, time.Now().Add(DefaultRecordTTL))
}

func (p *pubsubPublisher) PublishWithEOL(ctx context.Context, k ci.PrivKey, value path.Path, eol time.Time) error {
	id, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return err
	}

	_, ipnskey := IpnsKeysForID(id)

	seqno, err := p.getPreviousSeqNo(ctx, ipnskey)
	if err != nil {
		return err
	}

	seqno++

	return p.publishRecord(ctx, k, value, seqno, eol, ipnskey, id)
}

func (p *pubsubPublisher) getPreviousSeqNo(ctx context.Context, ipnskey string) (uint64, error) {
	// the datastore is shared with the routing publisher to properly increment and persist
	// ipns record sequence numbers.
	prevrec, err := p.ds.Get(dshelp.NewKeyFromBinary([]byte(ipnskey)))
	if err != nil {
		if err == ds.ErrNotFound {
			// None found, lets start at zero!
			return 0, nil
		}
		return 0, err
	}

	prbytes, ok := prevrec.([]byte)
	if !ok {
		return 0, fmt.Errorf("unexpected type returned from datastore: %#v", prevrec)
	}

	var dsrec dhtpb.Record
	err = proto.Unmarshal(prbytes, &dsrec)
	if err != nil {
		return 0, err
	}

	var entry pb.IpnsEntry
	err = proto.Unmarshal(dsrec.GetValue(), &entry)
	if err != nil {
		return 0, err
	}

	return entry.GetSequence(), nil
}

func (p *pubsubPublisher) publishRecord(ctx context.Context, k ci.PrivKey, value path.Path, seqno uint64, eol time.Time, ipnskey string, ID peer.ID) error {
	entry, err := CreateRoutingEntryData(k, value, seqno, eol)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	// the datastore is shared with the routing publisher to properly increment and persist
	// ipns record sequence numbers; so we need to Record our new entry in the datastore
	dsrec, err := record.MakePutRecord(k, ipnskey, data, true)
	if err != nil {
		return err
	}

	dsdata, err := proto.Marshal(dsrec)
	if err != nil {
		return err
	}

	p.ds.Put(dshelp.NewKeyFromBinary([]byte(ipnskey)), dsdata)

	// now we publish, but we also need to bootstrap pubsub for our messages to propagate
	topic := "/ipns/" + ID.Pretty()

	p.mx.Lock()
	_, ok := p.subs[topic]

	if !ok {
		p.subs[ipnskey] = struct{}{}
		p.mx.Unlock()

		bootstrapPubsub(p.ctx, p.cr, p.host, topic)
	} else {
		p.mx.Unlock()
	}

	log.Debugf("PubsubPublish: publish IPNS record for %s", topic)
	return p.ps.Publish(topic, data)
}

// PubsubResolver implementation
func (r *pubsubResolver) Resolve(ctx context.Context, name string) (value path.Path, err error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

func (r *pubsubResolver) ResolveN(ctx context.Context, name string, depth int) (value path.Path, err error) {
	return resolve(ctx, r, name, depth, "/ipns/")
}

func (r *pubsubResolver) resolveOnce(ctx context.Context, name string) (value path.Path, err error) {
	log.Debugf("PubsubResolve: '%s'", name)

	// retrieve the public key once (for verifying messages)
	xname := strings.TrimPrefix(name, "/ipns/")
	hash, err := mh.FromB58String(xname)
	if err != nil {
		log.Warningf("PubsubResolve: bad input hash: [%s]", xname)
		return
	}

	pubk, err := r.pkf.GetPublicKey(ctx, peer.ID(hash))
	if err != nil {
		log.Warningf("PubsubResolve: error fetching public key: %s [%s]", err.Error(), xname)
		return
	}

	// the mutex is locked both for subscription map manipulation and datastore
	// manipulation (otherwise race in TTL checking/invalidation)
	r.mx.Lock()
	defer r.mx.Unlock()

	// see if we already have a pubsub subscription; if not, subscribe
	sub, ok := r.subs[name]
	if !ok {
		sub, err = r.ps.Subscribe(name)
		if err != nil {
			return
		}

		r.subs[name] = sub
		go r.handleSubscription(sub, name, pubk)
		go bootstrapPubsub(r.ctx, r.cr, r.host, name)
	}

	// resolve to what we may already have in the datastore
	dsval, err := r.ds.Get(dshelp.NewKeyFromBinary([]byte(name)))
	if err != nil {
		// this signals ds.ErrNotFound when we have no record
		return
	}

	entry := dsval.(*pb.IpnsEntry)

	// check EOL; if the entry has expired, delete from datastore and return ds.ErrNotFound
	eol, ok := checkEOL(entry)
	if ok && eol.Before(time.Now()) {
		r.ds.Delete(dshelp.NewKeyFromBinary([]byte(name)))
		return "", ds.ErrNotFound
	}

	value, err = path.ParsePath(string(entry.GetValue()))
	return
}

func (r *pubsubResolver) handleSubscription(sub *floodsub.Subscription, name string, pubk ci.PubKey) {
	defer sub.Cancel()

	for {
		msg, err := sub.Next(r.ctx)
		if err != nil {
			log.Warningf("PubsubResolve: subscription error in %s: %s", name, err.Error())
			return
		}

		err = r.receive(msg, name, pubk)
		if err != nil {
			log.Debugf("PubsubResolve: error proessing update for %s: %s", name, err.Error())
		}
	}
}

func (r *pubsubResolver) receive(msg *floodsub.Message, name string, pubk ci.PubKey) error {
	data := msg.GetData()
	if data == nil {
		return errors.New("empty message")
	}

	entry := new(pb.IpnsEntry)
	err := proto.Unmarshal(data, entry)
	if err != nil {
		return err
	}

	ok, err := pubk.Verify(ipnsEntryDataForSig(entry), entry.GetSignature())
	if err != nil || !ok {
		return errors.New("signature verification failed")
	}

	_, err = path.ParsePath(string(entry.GetValue()))
	if err != nil {
		return err
	}

	eol, ok := checkEOL(entry)
	if ok && eol.Before(time.Now()) {
		return errors.New("stale update")
	}

	r.mx.Lock()
	defer r.mx.Unlock()

	// check the sequence number against what we may already have in our datastore
	dsval, err := r.ds.Get(dshelp.NewKeyFromBinary([]byte(name)))
	if err == nil {
		oentry := dsval.(*pb.IpnsEntry)
		if entry.GetSequence() <= oentry.GetSequence() {
			return errors.New("stale update")
		}
	}

	log.Debugf("PubsubResolve: receive IPNS record for %s", name)
	r.ds.Put(dshelp.NewKeyFromBinary([]byte(name)), entry)
	return nil
}

// rendezvous with peers in the name topic through provider records
// Note: rendezbous/boostrap should really be handled by the pubsub implementation itself!
func bootstrapPubsub(ctx context.Context, cr routing.ContentRouting, host p2phost.Host, name string) {
	topic := "floodsub:" + name
	hash := u.Hash([]byte(topic))
	rz := cid.NewCidV1(cid.Raw, hash) // perhaps this should be V0

	err := cr.Provide(ctx, rz, true)
	if err != nil {
		log.Warningf("bootstrapPubsub: Error providing rendezvous for %s: %s", topic, err.Error())
	}

	rzctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	wg := &sync.WaitGroup{}
	for pi := range cr.FindProvidersAsync(rzctx, rz, 10) {
		if pi.ID == host.ID() {
			continue
		}
		wg.Add(1)
		go func(pi pstore.PeerInfo) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			err := host.Connect(ctx, pi)
			if err != nil {
				log.Debugf("Error connecting to pubsub peer %s: %s", pi.ID, err.Error())
				return
			}
			log.Debugf("Connected to pubsub peer %s", pi.ID)
		}(pi)
	}

	wg.Wait()
}
