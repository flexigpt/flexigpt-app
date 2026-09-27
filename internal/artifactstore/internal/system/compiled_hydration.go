package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (c *Components) RegisterCompiledPackages(
	ctx context.Context,
	values []topology.CompiledRegistration,
) error {
	if c == nil || c.Refresh == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}

	for _, value := range values {
		if value.Files == nil {
			return fmt.Errorf(
				"%w: compiled built-in registration has no embedded file system",
				basespec.ErrInvalid,
			)
		}
		if !c.isProtectedRoot(value.Set.Hydration.RootID) {
			return fmt.Errorf(
				"%w: compiled built-in registration targets an unprotected Root",
				basespec.ErrProtected,
			)
		}

		// This is the runtime trust boundary. Catalog values are generated
		// from the ordinary parser and Store admission path, then compared
		// against that path in tests. Runtime only registers their source
		// locator and file digest witnesses.
		if err := c.Refresh.RegisterCompiledDocuments(
			ctx,
			value.Set.Hydration.RootID,
			value.Set.Hydration.SourceID,
			value.Set.Packages,
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *Components) HydrateCompiledPackages(
	ctx context.Context,
	plans []topology.CompiledPackagePlan,
) error {
	if c == nil ||
		c.SourceRuntime == nil ||
		c.Sources == nil ||
		c.Refresh == nil ||
		c.Artifacts == nil ||
		c.managedSources == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	type lifecyclePlan struct {
		plan  topology.CompiledPackagePlan
		state any
	}
	lifecyclePlans := make([]lifecyclePlan, 0, len(plans))
	for _, plan := range plans {
		var state any
		if plan.Registration.Lifecycle != nil {
			value, err := plan.Registration.Lifecycle.PrepareCompiledHydration(
				ctx,
				plan,
			)
			if err != nil {
				return err
			}
			state = value
		}
		lifecyclePlans = append(lifecyclePlans, lifecyclePlan{
			plan: plan, state: state,
		})
	}
	type sourceKey struct {
		rootID   root.RootID
		sourceID source.SourceID
	}
	type sourceBatch struct {
		publications []source.ManagedPackagePublication
		removals     []source.ManagedPackageAddress
		verify       []topology.CompiledPackage
	}

	batches := make(map[sourceKey]*sourceBatch)
	for _, lifecycle := range lifecyclePlans {
		plan := lifecycle.plan
		set := plan.Registration.Set
		key := sourceKey{
			rootID:   set.Hydration.RootID,
			sourceID: set.Hydration.SourceID,
		}

		batch := batches[key]
		if batch == nil {
			batch = &sourceBatch{}
			batches[key] = batch
		}

		byScope := make(map[basespec.Locator]topology.CompiledPackage)
		for _, packageValue := range set.Packages {
			scope, err := packageValue.Address.Directory()
			if err != nil {
				return err
			}
			byScope[scope] = packageValue
		}

		for _, scope := range plan.Changed {
			packageValue, found := byScope[scope]
			if !found {
				return fmt.Errorf(
					"%w: compiled hydration selected unknown package scope %q",
					basespec.ErrInvalid,
					scope,
				)
			}

			files, err := readCompiledPackageFiles(
				ctx,
				plan.Registration.Files,
				packageValue,
			)
			if err != nil {
				return err
			}
			batch.publications = append(
				batch.publications,
				source.ManagedPackagePublication{
					Address: packageValue.Address,
					Files:   files,
				},
			)
			batch.verify = append(batch.verify, packageValue)
		}

		for _, stale := range plan.Stale {
			address, err := source.ParseManagedPackageAddressDirectory(
				stale.Key.Scope,
			)
			if err != nil {
				return err
			}
			batch.removals = append(batch.removals, address)
		}
	}

	keys := make([]sourceKey, 0, len(batches))
	for key := range batches {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].rootID != keys[right].rootID {
			return keys[left].rootID < keys[right].rootID
		}
		return keys[left].sourceID < keys[right].sourceID
	})

	for _, key := range keys {
		batch := batches[key]
		if len(batch.publications) == 0 &&
			len(batch.removals) == 0 {
			continue
		}

		sourceValue, err := c.SourceRuntime.Get(
			ctx,
			key.rootID,
			key.sourceID,
		)
		if err != nil {
			return err
		}
		if sourceValue.Kind != source.SourceKindManagedDirectory ||
			!sourceValue.Enabled {
			return fmt.Errorf(
				"%w: compiled hydration requires an enabled managed Source",
				basespec.ErrInvalid,
			)
		}

		if err := c.managedSources.ApplyPackageBatch(
			ctx,
			sourceValue,
			batch.publications,
			batch.removals,
		); err != nil {
			return err
		}

		// The plan exists because a package hydration marker was absent or
		// stale. Advance Source metadata once even if the filesystem already
		// contains equivalent bytes after an interrupted previous startup.
		//
		// This gives the next refresh one coherent Source revision and avoids
		// per-package publication and refresh loops.
		if _, err := c.Sources.MarkContentChanged(
			ctx,
			key.rootID,
			key.sourceID,
			sourceValue.Revision,
		); err != nil {
			return err
		}

		if _, err := c.Refresh.RefreshSource(
			ctx,
			key.rootID,
			key.sourceID,
		); err != nil {
			return err
		}

		for _, packageValue := range batch.verify {
			if err := c.verifyCompiledPackage(
				ctx,
				key.rootID,
				key.sourceID,
				packageValue,
			); err != nil {
				return err
			}
		}
	}
	for _, lifecycle := range lifecyclePlans {
		if lifecycle.plan.Registration.Lifecycle == nil {
			continue
		}
		if err := lifecycle.plan.Registration.Lifecycle.CompleteCompiledHydration(
			ctx,
			lifecycle.plan,
			lifecycle.state,
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *Components) verifyCompiledPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	value topology.CompiledPackage,
) error {
	for _, document := range value.Documents {
		locator, err := value.Address.FileLocator(document.Locator)
		if err != nil {
			return err
		}
		for _, expected := range document.Artifacts {
			record, err := c.Artifacts.FindByOrigin(
				ctx,
				rootID,
				artifact.SourceBinding{
					SourceID:           sourceID,
					Locator:            locator,
					SubresourceLocator: expected.Subresource,
				},
				expected.Definition.Kind,
			)
			if err != nil {
				return err
			}
			if record.State != artifact.StateAvailable ||
				record.LogicalName != expected.Definition.LogicalName ||
				record.LogicalVersion != expected.Definition.LogicalVersion ||
				record.ResolvedDefinition == nil ||
				*record.ResolvedDefinition != expected.Definition.Digest ||
				record.SourceContentDigest == nil ||
				*record.SourceContentDigest != document.Digest {
				return fmt.Errorf(
					"%w: compiled package Artifact does not match generated catalog",
					basespec.ErrDigestMismatch,
				)
			}
		}
	}
	return nil
}

func readCompiledPackageFiles(
	ctx context.Context,
	filesystem fs.FS,
	value topology.CompiledPackage,
) ([]source.ManagedPackageFile, error) {
	if filesystem == nil {
		return nil, basespec.ErrInvalid
	}

	output := make([]source.ManagedPackageFile, 0, len(value.Files))
	for _, file := range value.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := file.Locator.ValidatePortable(false); err != nil {
			return nil, err
		}

		location := path.Join(
			string(value.EmbeddedRoot),
			string(file.Locator),
		)
		reader, err := filesystem.Open(location)
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(
			io.LimitReader(reader, file.Size+1),
		)
		closeErr := reader.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return nil, err
		}
		if int64(len(content)) != file.Size ||
			cryptoutil.DigestBytes(content) != file.Digest {
			return nil, fmt.Errorf(
				"%w: embedded file %q differs from generated catalog",
				basespec.ErrDigestMismatch,
				location,
			)
		}

		output = append(output, source.ManagedPackageFile{
			Locator: file.Locator,
			Content: content,
		})
	}
	return output, nil
}
