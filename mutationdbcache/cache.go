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

func List(assembly string) ([]*mutations.Dataset, error) {
	return instance.List(assembly)
}

func Dir() string {
	return instance.Dir()
}

func GetDataset(assembly string, uuid string) (*mutations.Dataset, error) {
	return instance.GetDataset(assembly, uuid)
}

func Search(assembly string, location *dna.Location, uuids []string) (*mutations.SearchResults, error) {
	return instance.Search(assembly, location, uuids)
}
