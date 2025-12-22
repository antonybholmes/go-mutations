package mutationdb

import (
	"sync"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-mutations"
)

var instance *mutations.MutationDB
var once sync.Once

func InitMutationDB(dir string) (*mutations.MutationDB, error) {
	once.Do(func() {
		instance = mutations.NewMutationDB(dir)
	})

	return instance, nil
}

func GetInstance() *mutations.MutationDB {
	return instance
}

func List(assembly string) ([]*mutations.Dataset, error) {
	return instance.ListDatasets(assembly)
}

func Dir() string {
	return instance.Dir()
}

func GetDataset(assembly string, publicId string) (*mutations.Dataset, error) {
	return instance.GetDataset(assembly, publicId)
}

func Search(assembly string, location *dna.Location, publicIds []string) (*mutations.SearchResults, error) {
	return instance.Search(assembly, location, publicIds)
}
