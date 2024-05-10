package mutationdb

import (
	"fmt"

	"github.com/antonybholmes/go-dna"
	"github.com/antonybholmes/go-mutations"
)

// pretend its a global const
var instance *mutations.MutationDB

func InitDB(path string, mutationSet *mutations.MutationSet) error {
	var err error

	instance, err = mutations.NewMutationDB(path, mutationSet)

	return err
}

func FindMutations(location *dna.Location) (*mutations.MutationResults, error) {
	if instance == nil {
		return nil, fmt.Errorf("mutation db not initialized")
	}

	return instance.FindMutations(location)
}
