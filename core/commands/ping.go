package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	cmds "github.com/ipfs/go-ipfs/commands"
	core "github.com/ipfs/go-ipfs/core"

	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	"gx/ipfs/QmVD1W3MC8Hk1WZgFQPWWmBECJ3X72BgUYf9eCQ4PGzPps/go-ipfs-cmdkit"
	ma "gx/ipfs/QmW8s4zTsUoX1Q6CeYxVKPyqSKbF7H1YDUyTostBtZ8DaG/go-multiaddr"
	peer "gx/ipfs/QmWNY7dV54ZDYmTA1ykVdwNCqC11mpU4zSUp6XDpLTH9eG/go-libp2p-peer"
	pstore "gx/ipfs/QmYijbtjCxFEjSXaudaQAUz3LN5VKLssm8WCUsRoqzXmQR/go-libp2p-peerstore"
)

const kPingTimeout = 10 * time.Second

type PingResult struct {
	Success bool
	Time    time.Duration
	Text    string
}

var PingCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Send echo request packets to IPFS hosts.",
		ShortDescription: `
'ipfs ping' is a tool to test sending data to other nodes. It finds nodes
via the routing system, sends pings, waits for pongs, and prints out round-
trip latency information.
		`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("peer ID", true, true, "ID of peer to be pinged.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.IntOption("count", "n", "Number of ping messages to send.").WithDefault(10),
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			v, err := unwrapOutput(res.Output())
			if err != nil {
				return nil, err
			}

			obj, ok := v.(*PingResult)
			if !ok {
				return nil, u.ErrCast()
			}

			buf := new(bytes.Buffer)
			if len(obj.Text) > 0 {
				buf = bytes.NewBufferString(obj.Text + "\n")
			} else if obj.Success {
				fmt.Fprintf(buf, "Pong received: time=%.2f ms\n", obj.Time.Seconds()*1000)
			} else {
				fmt.Fprintf(buf, "Pong failed\n")
			}
			return buf, nil
		},
	},
	Run: func(req cmds.Request, res cmds.Response) {
		ctx := req.Context()
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		// Must be online!
		if !n.OnlineMode() {
			res.SetError(errNotOnline, cmdkit.ErrClient)
			return
		}

		addr, peerID, err := ParsePeerParam(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		if addr != nil {
			n.Peerstore.AddAddr(peerID, addr, pstore.TempAddrTTL) // temporary
		}

		numPings, _, err := req.Option("count").Int()
		if err != nil {
			res.SetError(err, cmdkit.ErrNormal)
			return
		}

		outChan := pingPeer(ctx, n, peerID, numPings)
		res.SetOutput(outChan)
	},
	Type: PingResult{},
}

func pingPeer(ctx context.Context, n *core.IpfsNode, pid peer.ID, numPings int) <-chan interface{} {
	outChan := make(chan interface{})
	go func() {
		defer close(outChan)

		if len(n.Peerstore.Addrs(pid)) == 0 {
			// Make sure we can find the node in question
			outChan <- &PingResult{
				Text:    fmt.Sprintf("Looking up peer %s", pid.Pretty()),
				Success: true,
			}

			ctx, cancel := context.WithTimeout(ctx, kPingTimeout)
			defer cancel()
			p, err := n.Routing.FindPeer(ctx, pid)
			if err != nil {
				outChan <- &PingResult{Text: fmt.Sprintf("Peer lookup error: %s", err)}
				return
			}
			n.Peerstore.AddAddrs(p.ID, p.Addrs, pstore.TempAddrTTL)
		}

		outChan <- &PingResult{
			Text:    fmt.Sprintf("PING %s.", pid.Pretty()),
			Success: true,
		}

		ctx, cancel := context.WithTimeout(ctx, kPingTimeout*time.Duration(numPings))
		defer cancel()
		pings, err := n.Ping.Ping(ctx, pid)
		if err != nil {
			outChan <- &PingResult{
				Success: false,
				Text:    fmt.Sprintf("Ping error: %s", err),
			}
			return
		}

		var done bool
		var total time.Duration
		for i := 0; i < numPings && !done; i++ {
			select {
			case <-ctx.Done():
				done = true
				break
			case t, ok := <-pings:
				if !ok {
					done = true
					break
				}

				outChan <- &PingResult{
					Success: true,
					Time:    t,
				}
				total += t
				time.Sleep(time.Second)
			}
		}
		averagems := total.Seconds() * 1000 / float64(numPings)
		outChan <- &PingResult{
			Success: true,
			Text:    fmt.Sprintf("Average latency: %.2fms", averagems),
		}
	}()
	return outChan
}

func ParsePeerParam(text string) (ma.Multiaddr, peer.ID, error) {
	// to be replaced with just multiaddr parsing, once ptp is a multiaddr protocol
	idx := strings.LastIndex(text, "/")
	if idx == -1 {
		pid, err := peer.IDB58Decode(text)
		if err != nil {
			return nil, "", err
		}

		return nil, pid, nil
	}

	addrS := text[:idx]
	peeridS := text[idx+1:]

	var maddr ma.Multiaddr
	var pid peer.ID

	// make sure addrS parses as a multiaddr.
	if len(addrS) > 0 {
		var err error
		maddr, err = ma.NewMultiaddr(addrS)
		if err != nil {
			return nil, "", err
		}
	}

	// make sure idS parses as a peer.ID
	var err error
	pid, err = peer.IDB58Decode(peeridS)
	if err != nil {
		return nil, "", err
	}

	return maddr, pid, nil
}
