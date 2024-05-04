package mutationdb

import (
	"fmt"

	"github.com/antonybholmes/go-mutations"
)

// pretend its a global const
var instance *mutations.MutationDB

func InitDB(path string, mutationSet *mutations.MutationSet) error {
	var err error

	instance, err = mutations.NewMutationDB(path, mutationSet)

	return err
}

func FindSamples(search string) (*mutations.MutationResults, error) {
	if instance == nil {
		return nil, fmt.Errorf("microarray db not initialized")
	}

	return instance.FindSamples(search)
}
