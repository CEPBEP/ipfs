package commands

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	balanced "github.com/ipfs/go-ipfs/importer/balanced"
	"github.com/ipfs/go-ipfs/importer/chunk"
	ihelper "github.com/ipfs/go-ipfs/importer/helpers"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
)

var UrlStoreCmd = &cmds.Command{

	Subcommands: map[string]*cmds.Command{
		"add": urlAdd,
	},
}

var urlAdd = &cmds.Command{
	Arguments: []cmds.Argument{
		cmds.StringArg("url", true, false, "URL to add to IPFS"),
	},
	Type: BlockStat{},

	Run: func(req cmds.Request, res cmds.Response) {
		url := req.Arguments()[0]
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		hreq, err := http.NewRequest("GET", url, nil)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		hres, err := http.DefaultClient.Do(hreq)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		if hres.StatusCode != http.StatusOK {
			res.SetError(fmt.Errorf("expected code 200, got: %d", hres.StatusCode), cmds.ErrNormal)
			return
		}

		chk := chunk.NewSizeSplitter(hres.Body, chunk.DefaultBlockSize)
		dbp := &ihelper.DagBuilderParams{
			Dagserv:   n.DAG,
			RawLeaves: true,
			Maxlinks:  ihelper.DefaultLinksPerBlock,
			NoCopy:    true,
			Prefix: &cid.Prefix{
				Codec:    cid.DagProtobuf,
				MhLength: -1,
				MhType:   mh.SHA2_256,
				Version:  1,
			},
			URL: url,
		}

		blc, err := balanced.BalancedLayout(dbp.New(chk))
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(BlockStat{
			Key:  blc.Cid().String(),
			Size: int(hres.ContentLength),
		})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			bs := res.Output().(*BlockStat)
			return strings.NewReader(bs.Key + "\n"), nil
		},
	},
}
