package internal

import (
	"context"
	"fmt"

	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PackageBatchWriter is a trusted source-adapter capability used by generated
// built-in hydration.
//
// It stages complete package directories and publishes them package by package.
// The caller owns the single Source revision update and the final Source
// refresh after the batch completes.
type PackageBatchWriter interface {
	ApplyPackageBatch(
		ctx context.Context,
		value source.Source,
		publications []source.ManagedPackagePublication,
		removals []source.ManagedPackageAddress,
	) error
}

func (r *Registry) ApplyPackageBatch(
	ctx context.Context,
	value source.Source,
	publications []source.ManagedPackagePublication,
	removals []source.ManagedPackageAddress,
) error {
	if r == nil {
		return spec.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
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
	writer, supported := adapter.(PackageBatchWriter)
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
