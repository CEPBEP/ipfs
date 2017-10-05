package commands

import (
	"errors"
	"fmt"
	"io"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	ns "github.com/ipfs/go-ipfs/namesys"

	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
)

type ipnsPubsubState struct {
	Enabled bool
}

type ipnsPubsubCancel struct {
	Canceled bool
}

// IpnsPubsubCmd is the subcommand that allows us to manage the IPNS pubsub system
var IpnsPubsubCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "IPNS pubsub management",
		ShortDescription: `
Manage and inspect the state of the IPNS pubsub resolver.

Note: this command is experimental and subject to change as the system is refined
`,
	},
	Subcommands: map[string]*cmds.Command{
		"state":  ipnspsStateCmd,
		"subs":   ipnspsSubsCmd,
		"cancel": ipnspsCancelCmd,
	},
}

var ipnspsStateCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Query the state of IPNS pubsub",
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		_, ok := n.Namesys.GetResolver("pubsub")
		res.SetOutput(&ipnsPubsubState{ok})
	},
	Type: ipnsPubsubState{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			output, ok := res.Output().(*ipnsPubsubState)
			if !ok {
				return nil, u.ErrCast()
			}

			var state string
			if output.Enabled {
				state = "enabled"
			} else {
				state = "disabled"
			}

			return strings.NewReader(state + "\n"), nil
		},
	},
}

var ipnspsSubsCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Show current name subscriptions",
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		r, ok := n.Namesys.GetResolver("pubsub")
		if !ok {
			res.SetError(errors.New("IPNS pubsub subsystem is not enabled"), cmds.ErrClient)
			return
		}

		psr, ok := r.(*ns.PubsubResolver)
		if !ok {
			res.SetError(fmt.Errorf("unexpected resolver type: %v", r), cmds.ErrNormal)
			return
		}

		res.SetOutput(&stringList{psr.GetSubscriptions()})
	},
	Type: stringList{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
}

var ipnspsCancelCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Cancel a name subscription",
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		r, ok := n.Namesys.GetResolver("pubsub")
		if !ok {
			res.SetError(errors.New("IPNS pubsub subsystem is not enabled"), cmds.ErrClient)
			return
		}

		psr, ok := r.(*ns.PubsubResolver)
		if !ok {
			res.SetError(fmt.Errorf("unexpected resolver type: %v", r), cmds.ErrNormal)
			return
		}

		ok = psr.Cancel(req.Arguments()[0])
		res.SetOutput(&ipnsPubsubCancel{ok})
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("name", true, false, "Name to cancel the subscription for."),
	},
	Type: ipnsPubsubCancel{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			output, ok := res.Output().(*ipnsPubsubCancel)
			if !ok {
				return nil, u.ErrCast()
			}

			var state string
			if output.Canceled {
				state = "canceled"
			} else {
				state = "no subscription"
			}

			return strings.NewReader(state + "\n"), nil
		},
	},
}
