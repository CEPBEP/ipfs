package fsrepo

import (
	"fmt"
	"path"

	repo "github.com/ipfs/go-ipfs/repo"
	config "github.com/ipfs/go-ipfs/repo/config"
	"github.com/ipfs/go-ipfs/thirdparty/dir"

	ds "github.com/ipfs/go-datastore"
	mount "github.com/ipfs/go-datastore/syncmount"
	flatfs "github.com/ipfs/go-ds-flatfs"
	levelds "github.com/ipfs/go-ds-leveldb"
	measure "github.com/ipfs/go-ds-measure"
	ldbopts "github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	leveldbDirectory = "datastore"
	flatfsDirectory  = "blocks"
)

func openDefaultDatastore(r *FSRepo) (repo.Datastore, error) {
	leveldbPath := path.Join(r.path, leveldbDirectory)

	// save leveldb reference so it can be neatly closed afterward
	leveldbDS, err := levelds.NewDatastore(leveldbPath, &levelds.Options{
		Compression: ldbopts.NoCompression,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to open leveldb datastore: %v", err)
	}

	syncfs := !r.config.Datastore.NoSync

	// 2 characters of base32 suffix gives us 10 bits of freedom.
	// Leaving us with 10 bits, or 1024 way sharding
	blocksDS, err := flatfs.CreateOrOpen(path.Join(r.path, flatfsDirectory), flatfs.NextToLast(2), syncfs)
	if err != nil {
		return nil, fmt.Errorf("unable to open flatfs datastore: %v", err)
	}

	prefix := "ipfs.fsrepo.datastore."
	metricsBlocks := measure.New(prefix+"blocks", blocksDS)
	metricsLevelDB := measure.New(prefix+"leveldb", leveldbDS)
	mountDS := mount.New([]mount.Mount{
		{
			Prefix:    ds.NewKey("/blocks"),
			Datastore: metricsBlocks,
		},
		{
			Prefix:    ds.NewKey("/"),
			Datastore: metricsLevelDB,
		},
	})

	return mountDS, nil
}

func initDefaultDatastore(repoPath string, conf *config.Config) error {
	// The actual datastore contents are initialized lazily when Opened.
	// During Init, we merely check that the directory is writeable.
	leveldbPath := path.Join(repoPath, leveldbDirectory)
	if err := dir.Writable(leveldbPath); err != nil {
		return fmt.Errorf("datastore: %s", err)
	}

	flatfsPath := path.Join(repoPath, flatfsDirectory)
	if err := dir.Writable(flatfsPath); err != nil {
		return fmt.Errorf("datastore: %s", err)
	}
	return nil
}
