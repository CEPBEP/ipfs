package repo

import (
	"errors"
	"io"

	filestore "github.com/ipfs/go-ipfs/filestore"
	keystore "github.com/ipfs/go-ipfs/keystore"
	config "github.com/ipfs/go-ipfs/repo/config"

	ds "github.com/ipfs/go-datastore"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	ErrApiNotRunning = errors.New("api not running")
)

type Repo interface {
	Config() (*config.Config, error)
	SetConfig(*config.Config) error

	SetConfigKey(key string, value interface{}) error
	GetConfigKey(key string) (interface{}, error)

	Datastore() Datastore
	GetStorageUsage() (uint64, error)

	Keystore() keystore.Keystore

	FileManager() *filestore.FileManager

	// SetAPIAddr sets the API address in the repo.
	SetAPIAddr(addr ma.Multiaddr) error

	SwarmKey() ([]byte, error)

	io.Closer
}

// Datastore is the interface required from a datastore to be
// acceptable to FSRepo.
type Datastore interface {
	ds.Batching // should be threadsafe, just be careful
	io.Closer
}
