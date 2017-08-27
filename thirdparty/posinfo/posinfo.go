package posinfo

import (
	"os"

	node "github.com/ipfs/go-ipld-format"
)

type PosInfo struct {
	Offset   uint64
	FullPath string
	Stat     os.FileInfo // can be nil
}

type FilestoreNode struct {
	node.Node
	PosInfo *PosInfo
}
