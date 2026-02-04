package mutationdb

import (
	"sync"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-mutations"
)

var instance *mutations.MutationsDB
var once sync.Once

func InitMutationDB(dir string) (*mutations.MutationsDB, error) {
	once.Do(func() {
		instance = mutations.NewMutationsDB(dir)
	})

	return instance, nil
}

func GetInstance() *mutations.MutationsDB {
	return instance
}

func Datasets(assembly string, isAdmin bool, permissions []string) ([]*mutations.Dataset, error) {
	return instance.Datasets(assembly, isAdmin, permissions)
}

func Dir() string {
	return instance.Dir()
}

// func Dataset(datasetId string, isAdmin bool, permissions []string) (*mutations.Dataset, error) {
// 	return instance.Dataset(datasetId, isAdmin, permissions)
// }

func Search(assembly string, location *dna.Location, publicIds []string, isAdmin bool, permissions []string) (*mutations.SearchResults, error) {
	return instance.Search(assembly, location, publicIds, isAdmin, permissions)
}
