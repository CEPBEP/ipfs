package commands

import (
	"io"
	"strings"

	oldcmds "github.com/ipfs/go-ipfs/commands"
	dag "github.com/ipfs/go-ipfs/core/commands/dag"
	e "github.com/ipfs/go-ipfs/core/commands/e"
	files "github.com/ipfs/go-ipfs/core/commands/files"
	ocmd "github.com/ipfs/go-ipfs/core/commands/object"
	unixfs "github.com/ipfs/go-ipfs/core/commands/unixfs"

	lgc "github.com/ipfs/go-ipfs/commands/legacy"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	"gx/ipfs/QmVD1W3MC8Hk1WZgFQPWWmBECJ3X72BgUYf9eCQ4PGzPps/go-ipfs-cmdkit"
	"gx/ipfs/QmYopJAcV7R9SbxiPBCvqhnt8EusQpWPHewoZakCMt8hps/go-ipfs-cmds"
)

var log = logging.Logger("core/commands")

const (
	ApiOption = "api"
)

var Root = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:  "Global p2p merkle-dag filesystem.",
		Synopsis: "ipfs [--config=<config> | -c] [--debug=<debug> | -D] [--help=<help>] [-h=<h>] [--local=<local> | -L] [--api=<api>] <command> ...",
		Subcommands: `
BASIC COMMANDS
  init          Initialize ipfs local configuration
  add <path>    Add a file to IPFS
  cat <ref>     Show IPFS object data
  get <ref>     Download IPFS objects
  ls <ref>      List links from an object
  refs <ref>    List hashes of links from an object

DATA STRUCTURE COMMANDS
  block         Interact with raw blocks in the datastore
  object        Interact with raw dag nodes
  files         Interact with objects as if they were a unix filesystem
  dag           Interact with IPLD documents (experimental)

ADVANCED COMMANDS
  daemon        Start a long-running daemon process
  mount         Mount an IPFS read-only mountpoint
  resolve       Resolve any type of name
  name          Publish and resolve IPNS names
  key           Create and list IPNS name keypairs
  dns           Resolve DNS links
  pin           Pin objects to local storage
  repo          Manipulate the IPFS repository
  stats         Various operational stats
  p2p           Libp2p stream mounting
  filestore     Manage the filestore (experimental)

NETWORK COMMANDS
  id            Show info about IPFS peers
  bootstrap     Add or remove bootstrap peers
  swarm         Manage connections to the p2p network
  dht           Query the DHT for values or peers
  ping          Measure the latency of a connection
  diag          Print diagnostics

TOOL COMMANDS
  config        Manage configuration
  version       Show ipfs version information
  update        Download and apply go-ipfs updates
  commands      List all available commands

Use 'ipfs <command> --help' to learn more about each command.

ipfs uses a repository in the local file system. By default, the repo is
located at ~/.ipfs. To change the repo location, set the $IPFS_PATH
environment variable:

  export IPFS_PATH=/path/to/ipfsrepo

EXIT STATUS

The CLI will exit with one of the following values:

0     Successful execution.
1     Failed executions.
`,
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption("config", "c", "Path to the configuration file to use."),
		cmdkit.BoolOption("debug", "D", "Operate in debug mode."),
		cmdkit.BoolOption("help", "Show the full command help text."),
		cmdkit.BoolOption("h", "Show a short version of the command help text."),
		cmdkit.BoolOption("local", "L", "Run the command locally, instead of using the daemon."),
		cmdkit.StringOption(ApiOption, "Use a specific API instance (defaults to /ip4/127.0.0.1/tcp/5001)"),

		// global options, added to every command
		cmds.OptionEncodingType,
		cmds.OptionStreamChannels,
		cmds.OptionTimeout,
	},
}

// commandsDaemonCmd is the "ipfs commands" command for daemon
var CommandsDaemonCmd = CommandsCmd(Root)

var rootSubcommands = map[string]*cmds.Command{
	"add":       AddCmd,
	"bitswap":   BitswapCmd,
	"block":     BlockCmd,
	"cat":       CatCmd,
	"commands":  CommandsDaemonCmd,
	"filestore": FileStoreCmd,
	"get":       GetCmd,
	"pubsub":    PubsubCmd,
	"repo":      RepoCmd,
	"stats":     StatsCmd,
	"bootstrap": lgc.NewCommand(BootstrapCmd),
	"config":    lgc.NewCommand(ConfigCmd),
	"dag":       lgc.NewCommand(dag.DagCmd),
	"dht":       lgc.NewCommand(DhtCmd),
	"diag":      lgc.NewCommand(DiagCmd),
	"dns":       lgc.NewCommand(DNSCmd),
	"files":     lgc.NewCommand(files.FilesCmd),
	"id":        lgc.NewCommand(IDCmd),
	"key":       lgc.NewCommand(KeyCmd),
	"log":       lgc.NewCommand(LogCmd),
	"ls":        lgc.NewCommand(LsCmd),
	"mount":     lgc.NewCommand(MountCmd),
	"name":      lgc.NewCommand(NameCmd),
	"object":    lgc.NewCommand(ocmd.ObjectCmd),
	"pin":       lgc.NewCommand(PinCmd),
	"ping":      lgc.NewCommand(PingCmd),
	"p2p":       lgc.NewCommand(P2PCmd),
	"refs":      lgc.NewCommand(RefsCmd),
	"resolve":   lgc.NewCommand(ResolveCmd),
	"swarm":     lgc.NewCommand(SwarmCmd),
	"tar":       lgc.NewCommand(TarCmd),
	"file":      lgc.NewCommand(unixfs.UnixFSCmd),
	"update":    lgc.NewCommand(ExternalBinary()),
	"version":   lgc.NewCommand(VersionCmd),
	"shutdown":  lgc.NewCommand(daemonShutdownCmd),
}

// RootRO is the readonly version of Root
var RootRO = &cmds.Command{}

var CommandsDaemonROCmd = CommandsCmd(RootRO)

var RefsROCmd = &oldcmds.Command{}

var rootROSubcommands = map[string]*cmds.Command{
	"commands": CommandsDaemonROCmd,
	"cat":      CatCmd,
	"block": &cmds.Command{
		Subcommands: map[string]*cmds.Command{
			"stat": blockStatCmd,
			"get":  blockGetCmd,
		},
	},
	"get": GetCmd,
	"dns": lgc.NewCommand(DNSCmd),
	"ls":  lgc.NewCommand(LsCmd),
	"name": lgc.NewCommand(&oldcmds.Command{
		Subcommands: map[string]*oldcmds.Command{
			"resolve": IpnsCmd,
		},
	}),
	"object": lgc.NewCommand(&oldcmds.Command{
		Subcommands: map[string]*oldcmds.Command{
			"data":  ocmd.ObjectDataCmd,
			"links": ocmd.ObjectLinksCmd,
			"get":   ocmd.ObjectGetCmd,
			"stat":  ocmd.ObjectStatCmd,
		},
	}),
	"dag": lgc.NewCommand(&oldcmds.Command{
		Subcommands: map[string]*oldcmds.Command{
			"get":     dag.DagGetCmd,
			"resolve": dag.DagResolveCmd,
		},
	}),
	"refs":    lgc.NewCommand(RefsROCmd),
	"resolve": lgc.NewCommand(ResolveCmd),
	"version": lgc.NewCommand(VersionCmd),
}

func init() {
	Root.ProcessHelp()
	*RootRO = *Root

	// sanitize readonly refs command
	*RefsROCmd = *RefsCmd
	RefsROCmd.Subcommands = map[string]*oldcmds.Command{}

	Root.Subcommands = rootSubcommands

	RootRO.Subcommands = rootROSubcommands
}

type MessageOutput struct {
	Message string
}

func MessageTextMarshaler(res oldcmds.Response) (io.Reader, error) {
	v, err := unwrapOutput(res.Output())
	if err != nil {
		return nil, err
	}

	out, ok := v.(*MessageOutput)
	if !ok {
		return nil, e.TypeErr(out, v)
	}

	return strings.NewReader(out.Message), nil
}
