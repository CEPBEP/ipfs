package dagcmd

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	path "github.com/ipfs/go-ipfs/path"

	ipldcbor "gx/ipfs/QmNrbCt8j9DT5W9Pmjy2SdudT9k8GpaDr4sRuFix3BXhgR/go-ipld-cbor"
	btc "gx/ipfs/QmW3iy3akabBoGPriRX29Bsg698eMcZgQ17QYaFhpscCy7/go-ipld-btc"
	zec "gx/ipfs/QmYMG6BMmcMuEdLFtsNocXQE9ZWwCpoxNkZnvht1GHYRpu/go-ipld-zcash"
	cid "gx/ipfs/QmYhQaCYEcaPPjxJX7YcPcVKkQfRy6sJ7B3XmGFk82XYdQ/go-cid"
	node "gx/ipfs/Qmb3Hm9QDFmfYuET4pu7Kyg8JV78jFa1nvZx5vnCZsK4ck/go-ipld-format"
	eth "gx/ipfs/QmdtmRZK1C1EonaNnGTAz5DqPHNuKzPvuJ7Hf8Dg6sZMa4/go-ipld-eth"
)

var DagCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Interact with ipld dag objects.",
		ShortDescription: `
'ipfs dag' is used for creating and manipulating dag objects.

This subcommand is currently an experimental feature, but it is intended
to deprecate and replace the existing 'ipfs object' command moving forward.
		`,
	},
	Subcommands: map[string]*cmds.Command{
		"put": DagPutCmd,
		"get": DagGetCmd,
	},
}

type OutputObject struct {
	Cid *cid.Cid
}

var DagPutCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Add a dag node to ipfs.",
		ShortDescription: `
'ipfs dag put' accepts input from a file or stdin and parses it
into an object of the specified format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.FileArg("object data", true, false, "The object to put").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.StringOption("format", "f", "Format that the object will be added as.").Default("cbor"),
		cmds.StringOption("input-enc", "Format that the input object will be.").Default("json"),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		fi, err := req.Files().NextFile()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		ienc, _, _ := req.Option("input-enc").String()
		format, _, _ := req.Option("format").String()

		var outnodes []node.Node
		switch ienc {
		case "json":
			nd, err := convertJsonToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			outnodes = []node.Node{nd}
		case "hex":
			nds, err := convertHexToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			outnodes = nds
		case "raw":
			nds, err := convertRawToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			outnodes = nds
		default:
			res.SetError(fmt.Errorf("unrecognized input encoding: %s", ienc), cmds.ErrNormal)
			return
		}

		blkc, err := n.DAG.Add(outnodes[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
		}

		if len(outnodes) > 1 {
			b := n.DAG.Batch()
			for _, nd := range outnodes[1:] {
				_, err := b.Add(nd)
				if err != nil {
					res.SetError(err, cmds.ErrNormal)
					return
				}
			}
			if err := b.Commit(); err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
		}

		res.SetOutput(&OutputObject{Cid: blkc})
	},
	Type: OutputObject{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			oobj, ok := res.Output().(*OutputObject)
			if !ok {
				return nil, fmt.Errorf("expected a different object in marshaler")
			}

			return strings.NewReader(oobj.Cid.String()), nil
		},
	},
}

var DagGetCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Get a dag node from ipfs.",
		ShortDescription: `
'ipfs dag get' fetches a dag node from ipfs and prints it out in the specifed format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "The object to get").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, rem, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		var out interface{} = obj
		if len(rem) > 0 {
			final, _, err := obj.Resolve(rem)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			out = final
		}

		res.SetOutput(out)
	},
}

func convertJsonToType(r io.Reader, format string) (node.Node, error) {
	switch format {
	case "cbor", "dag-cbor":
		return ipldcbor.FromJson(r)
	case "dag-pb", "protobuf":
		return nil, fmt.Errorf("protobuf handling in 'dag' command not yet implemented")
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}

func convertHexToType(r io.Reader, format string) ([]node.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	decd, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	switch format {
	case "zec", "zcash":
		return zec.DecodeBlockMessage(decd)
	case "btc", "bitcoin":
		return btc.DecodeBlockMessage(decd)
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}

func convertRawToType(r io.Reader, format string) ([]node.Node, error) {
	switch format {
	case "eth":
		blk, txs, tries, uncles, err := eth.FromRlpBlockMessage(r)
		if err != nil {
			return nil, err
		}

		var out []node.Node
		out = append(out, blk)
		for _, tx := range txs {
			out = append(out, tx)
		}
		for _, t := range tries {
			out = append(out, t)
		}
		for _, unc := range uncles {
			out = append(out, unc)
		}
		return out, nil
	case "cbor", "dag-cbor":
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}

		nd, err := ipldcbor.Decode(data)
		if err != nil {
			return nil, err
		}

		return []node.Node{nd}, nil
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}
