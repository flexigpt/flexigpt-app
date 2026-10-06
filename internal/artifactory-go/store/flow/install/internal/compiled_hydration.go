package internal

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (c *Service) RegisterCompiledPackages(
	ctx context.Context,
	values []installModel.CompiledRegistration,
) error {
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}

	for _, value := range values {
		if !c.isProtectedRoot(value.Set.Hydration.RootID) {
			return fmt.Errorf(
				"%w: compiled built-in registration targets an unprotected Root",
				spec.ErrProtected,
			)
		}

		// This is the runtime trust boundary. Catalog values are generated
		// from the ordinary parser and Store admission path, then compared
		// against that path in tests. Runtime only registers their source
		// locator and file digest witnesses.
		documents, err := compiledDocumentsFromPackages(value.Set.Packages)
		if err != nil {
			return err
		}
		if err := c.refreshCompiled.RegisterCompiledDocuments(
			ctx,
			value.Set.Hydration.RootID,
			value.Set.Hydration.SourceID,
			documents,
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *Service) HydrateCompiledPackages(
	ctx context.Context,
	plans []installModel.CompiledPackagePlan,
) error {
	if c == nil ||
		c.SourceRuntime == nil ||
		c.Sources == nil ||
		c.Refresh == nil ||
		c.Artifacts == nil ||
		c.managedSources == nil {
		return spec.ErrClosed
	}
	if err := root.RequireInstallerPrivilege(ctx); err != nil {
		return err
	}
	type lifecyclePlan struct {
		plan  installModel.CompiledPackagePlan
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
		rootID   rootModel.RootID
		sourceID sourceModel.SourceID
	}
	type sourceBatch struct {
		publications []managedpackageModel.ManagedPackagePublication
		removals     []managedpackageModel.ManagedPackageAddress
		verify       []installModel.CompiledPackage
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

		byScope := make(map[spec.Locator]installModel.CompiledPackage)
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
					spec.ErrInvalid,
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
				managedpackageModel.ManagedPackagePublication{
					Address: packageValue.Address,
					Files:   files,
				},
			)
			batch.verify = append(batch.verify, packageValue)
		}

		for _, stale := range plan.Stale {
			address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
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
		if !c.managedSources.SupportsManagedPackages(sourceValue.Kind) ||
			!sourceValue.Enabled {
			return fmt.Errorf(
				"%w: compiled hydration requires an enabled managed Source",
				spec.ErrInvalid,
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
		if _, err := c.sourceContent.MarkContentChanged(
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

func (c *Service) verifyCompiledPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	value installModel.CompiledPackage,
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
				artifactModel.SourceBinding{
					SourceID:           sourceID,
					Locator:            locator,
					SubresourceLocator: expected.Subresource,
				},
				expected.Definition.Kind,
			)
			if err != nil {
				return err
			}
			if record.State != artifactModel.StateAvailable ||
				record.LogicalName != expected.Definition.LogicalName ||
				record.LogicalVersion != expected.Definition.LogicalVersion ||
				record.ResolvedDefinition == nil ||
				*record.ResolvedDefinition != expected.Definition.Digest ||
				record.SourceContentDigest == nil ||
				*record.SourceContentDigest != document.Digest {
				return fmt.Errorf(
					"%w: compiled package Artifact does not match generated catalog",
					spec.ErrDigestMismatch,
				)
			}
		}
	}
	return nil
}

func readCompiledPackageFiles(
	ctx context.Context,
	value installModel.CompiledPackage,
) ([]managedpackageModel.ManagedPackageFile, error) {
	if len(value.Files) == 0 || len(value.Files) > spec.MaxDiscoveryEntries {
		return nil, fmt.Errorf(
			"%w: compiled package has an invalid file count",
			spec.ErrInvalid,
		)
	}

	var total int64
	output := make([]managedpackageModel.ManagedPackageFile, 0, len(value.Files))
	for _, file := range value.Files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := file.Locator.ValidatePortable(false); err != nil {
			return nil, err
		}

		if file.Size < 0 || file.Size > spec.MaxScanBytes-total {
			return nil, fmt.Errorf(
				"%w: compiled package exceeds the byte limit",
				spec.ErrInvalid,
			)
		}
		actual := cryptoutil.DigestBytes(file.Content)
		if int64(len(file.Content)) != file.Size || actual != file.Digest {
			return nil, fmt.Errorf(
				"%w: compiled package %q file %q: size %d/%d, digest %q/%q",
				spec.ErrDigestMismatch,
				value.Address,
				file.Locator,
				len(file.Content),
				file.Size,
				actual,
				file.Digest,
			)
		}
		total += file.Size

		output = append(output, managedpackageModel.ManagedPackageFile{
			Locator: file.Locator,
			Content: append([]byte(nil), file.Content...),
		})
	}
	return output, nil
}
