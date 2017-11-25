// package importer implements utilities used to create IPFS DAGs from files
// and readers
package importer

import (
	"fmt"
	"os"

	"gx/ipfs/QmUyfy4QSr3NXym4etEiRyxBLqqAeKHJuRdi8AACxg63fZ/go-ipfs-cmdkit/files"

	bal "github.com/ipfs/go-ipfs/importer/balanced"
	"github.com/ipfs/go-ipfs/importer/chunk"
	h "github.com/ipfs/go-ipfs/importer/helpers"
	trickle "github.com/ipfs/go-ipfs/importer/trickle"
	dag "github.com/ipfs/go-ipfs/merkledag"

	node "gx/ipfs/QmPN7cwmpcc4DWXb4KTB9dNAJgjuPY69h3npsMfhRrQL9c/go-ipld-format"

	"github.com/ipfs/go-ipfs/providers"
)

// Builds a DAG from the given file, writing created blocks to disk as they are
// created
func BuildDagFromFile(fpath string, ds dag.DAGService, prov providers.Interface) (node.Node, error) {
	stat, err := os.Lstat(fpath)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("`%s` is a directory", fpath)
	}

	f, err := files.NewSerialFile(fpath, fpath, false, stat)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return BuildDagFromReader(ds, prov, chunk.NewSizeSplitter(f, chunk.DefaultBlockSize))
}

// BuildDagFromReader creates new DAG containing data provided by Splitter
func BuildDagFromReader(ds dag.DAGService, prov providers.Interface, spl chunk.Splitter) (node.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Provider: prov,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return bal.BalancedLayout(dbp.New(spl))
}

// BuildTrickleDagFromReader creates new DAG with trickle layout containing data provided by Splitter
func BuildTrickleDagFromReader(ds dag.DAGService, prov providers.Interface, spl chunk.Splitter) (node.Node, error) {
	dbp := h.DagBuilderParams{
		Dagserv:  ds,
		Provider: prov,
		Maxlinks: h.DefaultLinksPerBlock,
	}

	return trickle.TrickleLayout(dbp.New(spl))
}
