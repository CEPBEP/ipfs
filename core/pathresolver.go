package core

import (
	"context"
	"errors"
	"strings"

	namesys "github.com/ipfs/go-ipfs/namesys"
	path "github.com/ipfs/go-ipfs/path"

	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	logging "gx/ipfs/QmeMFK13EDShD6L5dvrE9TEiuqtqRMBfxpgNkGxUYY4Vk5/go-log"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// ErrNoNamesys is an explicit error for when an IPFS node doesn't
// (yet) have a name system
var ErrNoNamesys = errors.New(
	"core/resolve: no Namesys on IpfsNode - can't resolve ipns entry")

// Resolve resolves the given path by parsing out protocol-specific
// entries (e.g. /ipns/<node-key>) and then going through the /ipfs/
// entries and returning the final node.
func Resolve(ctx context.Context, nsys namesys.NameSystem, r *path.Resolver, p path.Path) (node node.Node, err error) {
	eip := log.EventBegin(ctx, "Resolve")
	defer func() {
		eip.Append(logging.LoggableMap{"path": p.String()})
		if err != nil {
			eip.SetError(err)
		}
		eip.Done()
	}()
	if strings.HasPrefix(p.String(), "/ipns/") {
		// resolve ipns paths

		// TODO(cryptix): we sould be able to query the local cache for the path
		if nsys == nil {
			return nil, ErrNoNamesys
		}

		seg := p.Segments()

		if len(seg) < 2 || seg[1] == "" { // just "/<protocol/>" without further segments
			return nil, path.ErrNoComponents
		}

		extensions := seg[2:]
		resolvable, err := path.FromSegments("/", seg[0], seg[1])
		if err != nil {
			return nil, err
		}

		respath, err := nsys.Resolve(ctx, resolvable.String())
		if err != nil {
			return nil, err
		}

		segments := append(respath.Segments(), extensions...)
		p, err = path.FromSegments("/", segments...)
		if err != nil {
			return nil, err
		}
	}

	// ok, we have an IPFS path now (or what we'll treat as one)
	return r.ResolvePath(ctx, p)
}

// ResolveToCid resolves a path to a cid.
//
// It first checks if the path is already in the form of just a cid (<cid> or
// /ipfs/<cid>) and returns immediately if so. Otherwise, it falls back onto
// Resolve to perform resolution of the dagnode being referenced.
func ResolveToCid(ctx context.Context, nsys namesys.NameSystem, r *path.Resolver, p path.Path) (*cid.Cid, error) {

	// If the path is simply a cid, parse and return it. Parsed paths are already
	// normalized (read: prepended with /ipfs/ if needed), so segment[1] should
	// always be the key.
	if p.IsJustAKey() {
		return cid.Decode(p.Segments()[1])
	}

	// Fall back onto regular dagnode resolution. Retrieve the second-to-last
	// segment of the path and resolve its link to the last segment.
	head, tail, err := p.PopLastSegment()
	if err != nil {
		return nil, err
	}
	dagnode, err := Resolve(ctx, nsys, r, head)
	if err != nil {
		return nil, err
	}

	// Extract and return the cid of the link to the target dag node.
	link, _, err := dagnode.ResolveLink([]string{tail})
	if err != nil {
		return nil, err
	}

	return link.Cid, nil
}
