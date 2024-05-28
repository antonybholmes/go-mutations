package mutationdb

import (
	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-mutations"
)

// pretend its a global const
var instance *mutations.MutationDB

func InitDB(path string, metadata *mutations.MutationDBMetadata) error {
	var err error

	instance, err = mutations.NewMutationDB(path, metadata)

	return err
}

func FindMutations(location *dna.Location) (*mutations.MutationResults, error) {
	return instance.FindMutations(location)
}

func Pileup(location *dna.Location) (*mutations.Pileup, error) {
	return instance.Pileup(location)
}
