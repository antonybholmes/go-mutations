package mutationdb

import (
	"fmt"

	"github.com/antonybholmes/go-mutations"
)

// pretend its a global const
var instance *mutations.MutationsDB

func InitDB(path string) error {
	var err error

	instance, err = mutations.NewMutationDB(path)

	return err
}

func FindSamples(array string, search string) (*[]mutations.Mutation, error) {
	if instance == nil {
		return nil, fmt.Errorf("microarray db not initialized")
	}

	return instance.FindSamples(array, search)
}

func Expression(samples *mutations.MutationsReq) (*mutations.ExpressionData, error) {
	if instance == nil {
		return nil, fmt.Errorf("microarray db not initialized")
	}

	return instance.Expression(samples)
}
