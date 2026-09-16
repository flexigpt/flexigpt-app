package server

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SecretResolver interface {
	ResolveSecret(
		ctx context.Context,
		ref string,
	) (string, error)
}

type EnvironmentResolver interface {
	ResolveEnvironment(
		ctx context.Context,
		name string,
	) (string, bool, error)
}

type Resolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (Resolved, error)
}

func (r Resolved) MaterializeForInspection(
	ctx context.Context,
	environment EnvironmentResolver,
) (MaterializedServer, error) {
	if err := r.Validate(); err != nil {
		return MaterializedServer{}, err
	}
	return materializeValidated(
		ctx,
		r.Server,
		r.Document,
		r.Installation,
		nil,
		environment,
		false,
	)
}

func (r Resolved) MaterializeTrusted(
	ctx context.Context,
	secrets SecretResolver,
	environment EnvironmentResolver,
) (MaterializedServer, error) {
	return materializeValidated(
		ctx,
		r.Server,
		r.Document,
		r.Installation,
		secrets,
		environment,
		true,
	)
}

func (r Resolved) Validate() error {
	if err := r.Server.Validate(); err != nil {
		return err
	}
	if r.ArtifactRevision == 0 ||
		r.InstallationRevision == 0 {
		return fmt.Errorf(
			"%w: resolved MCP revisions are required",
			basespec.ErrInvalid,
		)
	}

	if err := cryptoutil.ValidateDigest(r.DefinitionDigest); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(r.SourceContentDigest); err != nil {
		return err
	}
	if err := basespec.ValidateSourceGeneration(r.SourceGeneration); err != nil {
		return err
	}
	if err := r.Document.Validate(); err != nil {
		return err
	}
	if err := r.Installation.Validate(); err != nil {
		return err
	}
	if err := r.Policy.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(r.Version)
}
