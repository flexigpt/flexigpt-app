package source

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (r *Registry) ApplyPackageBatch(
	ctx context.Context,
	value sourceModel.Source,
	publications []managedpackageModel.ManagedPackagePublication,
	removals []managedpackageModel.ManagedPackageAddress,
) error {
	if err := value.ValidateRead(); err != nil {
		return err
	}

	adapter, found := r.adapter(value.Kind)
	if !found {
		return fmt.Errorf(
			"%w: source adapter %q",
			spec.ErrSourceUnavailable,
			value.Kind,
		)
	}

	writer, supported := adapter.(managedpackage.BatchWriter)
	if !supported {
		return fmt.Errorf(
			"%w: source kind %q has no package batch writer",
			spec.ErrUnsupported,
			value.Kind,
		)
	}

	for _, publication := range publications {
		if err := publication.Address.Validate(); err != nil {
			return err
		}
		if publication.ExpectedGeneration != "" {
			return fmt.Errorf(
				"%w: compiled package batches do not use per-package generations",
				spec.ErrInvalid,
			)
		}
	}

	for _, address := range removals {
		if err := address.Validate(); err != nil {
			return err
		}
	}

	return writer.ApplyPackageBatch(
		ctx,
		value.Clone(),
		publications,
		removals,
	)
}
