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

	normalizedPublications := make(
		[]managedpackageModel.ManagedPackagePublication,
		len(publications),
	)
	seenPublications := make(
		map[managedpackageModel.ManagedPackageAddress]struct{},
		len(publications),
	)
	for index, publication := range publications {
		normalized, err := managedpackageModel.NormalizeManagedPackagePublication(
			publication,
		)
		if err != nil {
			return err
		}
		if normalized.ExpectedGeneration != "" {
			return fmt.Errorf(
				"%w: compiled package batches do not use per-package generations",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seenPublications[normalized.Address]; duplicate {
			return fmt.Errorf(
				"%w: compiled package batch repeats publication %v",
				spec.ErrConflict,
				normalized.Address,
			)
		}
		seenPublications[normalized.Address] = struct{}{}
		normalizedPublications[index] = normalized
	}

	normalizedRemovals := append(
		[]managedpackageModel.ManagedPackageAddress(nil),
		removals...,
	)
	for _, address := range normalizedRemovals {
		if err := address.Validate(); err != nil {
			return err
		}
	}

	return writer.ApplyPackageBatch(
		ctx,
		value.Clone(),
		normalizedPublications,
		normalizedRemovals,
	)
}
