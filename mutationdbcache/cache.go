package mutationdbcache

import (
	"sync"

	"github.com/antonybholmes/go-mutations"
)

var instance *mutations.MutationDBCache
var once sync.Once

func InitCache(dir string) (*mutations.MutationDBCache, error) {
	once.Do(func() {
		instance = mutations.NewMutationDBCache(dir)
	})

	return instance, nil
}

func GetInstance() *mutations.MutationDBCache {
	return instance
}

func Dir() string {
	return instance.Dir()
}

func MutationDB(mutationSet *mutations.MutationSet) (*mutations.MutationDB, error) {
	return instance.MutationDB(mutationSet)
}
