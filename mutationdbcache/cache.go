package mutationdbcache

import (
	"sync"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-mutations"
)

var instance *mutations.DatasetCache
var once sync.Once

func InitCache(dir string) (*mutations.DatasetCache, error) {
	once.Do(func() {
		instance = mutations.NewMutationDBCache(dir)
	})

	return instance, nil
}

func GetInstance() *mutations.DatasetCache {
	return instance
}

func Dir() string {
	return instance.Dir()
}

func GetDataset(uuid string) (*mutations.Dataset, error) {
	return instance.GetDataset(uuid)
}

func Search(location *dna.Location, uuids []string) (*mutations.SearchResults, error) {
	return instance.Search(location, uuids)
}
