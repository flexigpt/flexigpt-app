package system

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (c *Components) RegisterCompiledPackages(
	ctx context.Context,
	values []topology.CompiledRegistration,
) error {
	if c == nil || c.Refresh == nil {
		return model.ErrClosed
	}
	if err := install.RequirePrivileged(ctx); err != nil {
		return err
	}

	for _, value := range values {
		if !c.isProtectedRoot(value.Set.Hydration.RootID) {
			return fmt.Errorf(
				"%w: compiled built-in registration targets an unprotected Root",
				model.ErrProtected,
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
		return model.ErrClosed
	}
	if err := install.RequirePrivileged(ctx); err != nil {
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

		byScope := make(map[model.Locator]topology.CompiledPackage)
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
					model.ErrInvalid,
					scope,
				)
			}

			files, err := readCompiledPackageFiles(
				ctx,
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
				model.ErrInvalid,
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
					model.ErrDigestMismatch,
				)
			}
		}
	}
	return nil
}

func readCompiledPackageFiles(
	ctx context.Context,
	value topology.CompiledPackage,
) ([]source.ManagedPackageFile, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: compiled package context is nil", model.ErrInvalid)
	}
	if len(value.Files) == 0 || len(value.Files) > model.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: compiled package has an invalid file count",
			model.ErrInvalid,
		)
	}

	var total int64
	output := make([]source.ManagedPackageFile, 0, len(value.Files))
	for _, file := range value.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := file.Locator.ValidatePortable(false); err != nil {
			return nil, err
		}

		if file.Size < 0 || file.Size > model.MaxScanBytes-total {
			return nil, fmt.Errorf(
				"%w: compiled package exceeds the byte limit",
				model.ErrInvalid,
			)
		}
		actual := cryptoutil.DigestBytes(file.Content)
		if int64(len(file.Content)) != file.Size || actual != file.Digest {
			return nil, fmt.Errorf(
				"%w: compiled package %q file %q: size %d/%d, digest %q/%q",
				model.ErrDigestMismatch,
				value.Address,
				file.Locator,
				len(file.Content),
				file.Size,
				actual,
				file.Digest,
			)
		}
		total += file.Size

		output = append(output, source.ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		})
	}
	return output, nil
}
