package artifactstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Store) ScanRoot(ctx context.Context, rootID spec.RootID, plan spec.ScanPlan) (spec.ScanResult, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ScanResult{}, err
	}
	defer finish()

	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	root, err := s.repository.GetRoot(ctx, rootID, false)
	if err != nil {
		return spec.ScanResult{}, err
	}
	if !root.Enabled {
		return spec.ScanResult{}, fmt.Errorf("%w: root %q is disabled", spec.ErrConflict, rootID)
	}
	attachments, err := s.repository.ListRootSourceAttachments(ctx, rootID)
	if err != nil {
		return spec.ScanResult{}, err
	}
	attached := make(map[spec.SourceID]spec.RootSourceAttachment, len(attachments))
	sources := make(map[spec.SourceID]spec.ArtifactSource, len(attachments))
	sourceExpectations := make([]spec.RootScanSourceExpectation, 0, len(attachments))
	catalogByOccurrence := make(map[string]spec.CatalogResource)
	sourceVersions := map[spec.SourceID]spec.SourceCatalogVersion{}
	for _, attachment := range attachments {
		attached[attachment.SourceID] = attachment
		source, err := s.repository.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return spec.ScanResult{}, err
		}
		sources[source.SourceID] = source
		sourceExpectations = append(sourceExpectations, spec.RootScanSourceExpectation{
			SourceID:            source.SourceID,
			ObservationRevision: source.ObservationRevision,
			Enabled:             source.Enabled,
		})
		if !attachment.Enabled || !source.Enabled {
			continue
		}
		if source.LastObservedGeneration != nil {
			sourceVersions[source.SourceID] = spec.SourceCatalogVersion{
				Generation:          *source.LastObservedGeneration,
				ObservationRevision: source.ObservationRevision,
			}
		}
		resources, err := s.repository.ListCatalogResourcesForSource(ctx, source.SourceID)
		if err != nil {
			return spec.ScanResult{}, err
		}
		for _, resource := range resources {
			catalogByOccurrence[recordOccurrenceKey(
				resource.SourceID,
				resource.Locator,
				resource.SubresourceLocator,
			)] = resource
		}
	}

	plans := make(map[spec.SourceID]spec.SourceScanPlan, len(plan.SourcePlans))
	for _, sourcePlan := range plan.SourcePlans {
		if sourcePlan.SourceID == "" {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: scan source plan has an empty source ID",
				spec.ErrInvalidRequest,
			)
		}
		if _, exists := plans[sourcePlan.SourceID]; exists {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: duplicate source plan for %q",
				spec.ErrInvalidRequest,
				sourcePlan.SourceID,
			)
		}
		attachment, ok := attached[sourcePlan.SourceID]
		if !ok {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: source %q is not attached to root %q",
				spec.ErrSourceNotAttached,
				sourcePlan.SourceID,
				rootID,
			)
		}
		if !attachment.Enabled {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: source %q attachment is disabled for root %q",
				spec.ErrConflict,
				sourcePlan.SourceID,
				rootID,
			)
		}
		source, ok := sources[sourcePlan.SourceID]
		if !ok || !source.Enabled {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: source %q is disabled for root %q",
				spec.ErrConflict,
				sourcePlan.SourceID,
				rootID,
			)
		}
		if err := s.validateSourceScanPlan(sourcePlan); err != nil {
			return spec.ScanResult{}, err
		}
		plans[sourcePlan.SourceID] = sourcePlan
	}
	hasExplicitSourcePlans := len(plan.SourcePlans) > 0

	result := spec.ScanResult{RootID: rootID}
	executedPlan := spec.ScanPlan{}
	sourcePublications := make([]spec.SourceCatalogPublication, 0, len(attachments))
	for _, attachment := range attachments {
		if !attachment.Enabled {
			continue
		}
		source := sources[attachment.SourceID]
		if !source.Enabled {
			continue
		}
		sourcePlan, ok := plans[source.SourceID]
		if hasExplicitSourcePlans && !ok {
			if source.LastObservedGeneration == nil {
				return spec.ScanResult{}, fmt.Errorf(
					"%w: active source %q has never been observed and requires a source plan",
					spec.ErrInvalidRequest,
					source.SourceID,
				)
			}
			continue
		}
		if !hasExplicitSourcePlans {
			sourcePlan = spec.SourceScanPlan{
				SourceID:       source.SourceID,
				DirectoryRoots: []spec.DirectoryScanRoot{{Root: ".", Recursive: true}},
				MaxFileBytes:   spec.MaxDefinitionJSONBytes,
				Authoritative:  true,
			}
		}
		if err := s.validateSourceScanPlan(sourcePlan); err != nil {
			return spec.ScanResult{}, err
		}
		executedPlan.SourcePlans = append(executedPlan.SourcePlans, sourcePlan)
		sourceResult, publication, err := s.scanSource(ctx, source, sourcePlan)
		if err != nil {
			return spec.ScanResult{}, err
		}
		nextObservationRevision := source.ObservationRevision
		if publication.AdvanceObservationRevision {
			if nextObservationRevision >= spec.MaxObservationRevision {
				return spec.ScanResult{}, fmt.Errorf(
					"%w: source observation revision is exhausted",
					spec.ErrConflict,
				)
			}
			nextObservationRevision++
		}
		sourcePublications = append(sourcePublications, publication)
		applySourceCatalogPublication(catalogByOccurrence, publication)
		result.Sources = append(result.Sources, sourceResult)
		sourceVersions[source.SourceID] = spec.SourceCatalogVersion{
			Generation:          sourceResult.Generation,
			ObservationRevision: nextObservationRevision,
		}
		result.Diagnostics = appendBoundedDiagnostics(
			result.Diagnostics,
			sourceResult.Diagnostics...,
		)
	}

	// Confirm all scanned sources after every source has been scanned. Confirming
	// one source before scanning later sources leaves an avoidable publication
	// window in which an earlier source can change unnoticed.
	for _, sourceResult := range result.Sources {
		source := sources[sourceResult.SourceID]
		driver, ok := s.driverFor(source.Kind)
		if !ok {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: source kind %q",
				spec.ErrDriverUnavailable,
				source.Kind,
			)
		}
		confirmedGeneration, err := driver.Snapshot(ctx, source)
		if err != nil {
			return spec.ScanResult{}, err
		}
		if confirmedGeneration != sourceResult.Generation {
			return spec.ScanResult{}, fmt.Errorf(
				"%w: source %q changed while the root was being scanned",
				spec.ErrConflict,
				source.SourceID,
			)
		}
	}
	resources := catalogResourcesFromMap(catalogByOccurrence)
	planDigest, err := digestScanPlan(executedPlan)
	if err != nil {
		return spec.ScanResult{}, err
	}
	catalogDigest, err := digestCatalog(resources)
	if err != nil {
		return spec.ScanResult{}, err
	}
	generation, err := s.repository.PublishRootScan(
		ctx,
		spec.RootScanPublication{
			RootCatalog: spec.RootCatalogPublication{
				RootID:         rootID,
				RootRevision:   root.MountRevision,
				SourceVersions: sourceVersions,
				ScanPlanDigest: planDigest,
				CatalogDigest:  catalogDigest,
				CreatedAt:      s.nowUTC(),
				Diagnostics:    result.Diagnostics,
			},
			ExpectedRootRevision: root.MountRevision,
			Sources:              sourceExpectations,
			SourceCatalogs:       sourcePublications,
			CatalogResources:     resources,
		},
	)
	if err != nil {
		return spec.ScanResult{}, err
	}
	result.Generation = generation
	return result, nil
}

func (s *Store) scanSource(
	ctx context.Context,
	source spec.ArtifactSource,
	plan spec.SourceScanPlan,
) (spec.SourceScanResult, spec.SourceCatalogPublication, error) {
	driver, ok := s.driverFor(source.Kind)
	if !ok {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, fmt.Errorf(
			"%w: source kind %q",
			spec.ErrDriverUnavailable,
			source.Kind,
		)
	}
	existingResources, err := s.repository.ListCatalogResourcesForSource(ctx, source.SourceID)
	if err != nil {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, err
	}
	existingByLocator := make(map[spec.SourceLocator][]spec.CatalogResource)
	for _, resource := range existingResources {
		existingByLocator[resource.Locator] = append(
			existingByLocator[resource.Locator],
			resource,
		)
	}
	generation, err := driver.Snapshot(ctx, source)
	if err != nil {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, err
	}
	entries, err := collectSourceCandidates(ctx, driver, source, plan)
	if err != nil {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, err
	}
	now := s.nextModifiedAt(source.ModifiedAt)
	publication := spec.SourceCatalogPublication{
		SourceID:                    source.SourceID,
		ExpectedObservationRevision: source.ObservationRevision,
		ObservedGeneration:          generation,
		ObservedAt:                  now,
	}
	result := spec.SourceScanResult{SourceID: source.SourceID, Generation: generation}
	seenResources := make(map[string]struct{})
	for _, entry := range entries {
		result.Candidates++
		maximum := plan.MaxFileBytes
		if maximum <= 0 {
			maximum = spec.MaxDefinitionJSONBytes
		}
		if entry.SizeBytes < 0 || entry.SizeBytes > maximum {
			diagnostics := []spec.Diagnostic{{
				Severity: spec.DiagnosticSeverityWarning,
				Code:     "artifactstore.candidate.too-large",
				Message: fmt.Sprintf(
					"candidate exceeds the configured %d byte limit and was skipped",
					maximum,
				),
				Location: &spec.DiagnosticLocation{Locator: entry.Locator},
			}}
			invalid := invalidCandidateResources(
				source.SourceID,
				entry.Locator,
				"",
				now,
				nil,
				diagnostics,
				catalogResourcesForFrontendScope(
					existingByLocator[entry.Locator],
					plan.AllowedFrontendIDs,
				),
			)
			publication.Resources = append(publication.Resources, invalid...)
			result.InvalidResources += len(invalid)
			result.Diagnostics = appendBoundedDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		content, err := readCandidate(ctx, driver, source, entry, maximum)
		if err != nil {
			return result, publication, err
		}
		digest := baseutils.DigestBytes(content)
		candidate := spec.ArtifactCandidate{
			Source:              source,
			Locator:             entry.Locator,
			SourceContentDigest: digest,
			Content:             content,
		}
		frontend, err := s.selectFrontend(
			ctx,
			candidate,
			plan.AllowedFrontendIDs,
		)
		if err != nil {
			return result, publication, err
		}
		if frontend == nil {
			prior := catalogResourcesForFrontendScope(
				existingByLocator[entry.Locator],
				plan.AllowedFrontendIDs,
			)
			if len(prior) > 0 {
				diagnostics := []spec.Diagnostic{{
					Severity: spec.DiagnosticSeverityError,
					Code:     "artifactstore.frontend.unrecognized",
					Message:  "a previously cataloged source file is no longer recognized by an allowed frontend",
					Location: &spec.DiagnosticLocation{Locator: entry.Locator},
				}}
				invalid := invalidCandidateResources(
					source.SourceID,
					entry.Locator,
					"",
					now,
					&digest,
					diagnostics,
					prior,
				)
				publication.Resources = append(publication.Resources, invalid...)
				result.InvalidResources += len(invalid)
				result.Diagnostics = appendBoundedDiagnostics(
					result.Diagnostics,
					diagnostics...,
				)
			}
			continue
		}
		decoded, diagnostics := frontend.Decode(ctx, candidate)
		if len(decoded) == 0 {
			diagnostics = append(diagnostics, spec.Diagnostic{
				Severity: spec.DiagnosticSeverityError,
				Code:     "artifactstore.frontend.decode-empty",
				Message:  "the selected frontend emitted no definitions",
				Location: &spec.DiagnosticLocation{Locator: entry.Locator},
			})
		}
		if err := validate.ValidateDiagnostics(diagnostics); err != nil {
			return result, publication, fmt.Errorf(
				"%w: frontend %q returned invalid diagnostics: %w",
				spec.ErrInvalidRequest,
				frontend.ID(),
				err,
			)
		}
		if err := errorDiagnostics("frontend decode", diagnostics); err != nil || len(decoded) == 0 {
			invalid := invalidCandidateResources(
				source.SourceID,
				entry.Locator,
				frontend.ID(),
				now,
				&digest,
				diagnostics,
				catalogResourcesForFrontendScope(
					existingByLocator[entry.Locator],
					plan.AllowedFrontendIDs,
				),
			)
			publication.Resources = append(publication.Resources, invalid...)
			result.InvalidResources += len(invalid)
			result.Diagnostics = appendBoundedDiagnostics(
				result.Diagnostics,
				diagnostics...,
			)
			continue
		}
		for _, decodedArtifact := range decoded {
			resourceKey := string(entry.Locator) + "\x00" + string(decodedArtifact.SubresourceLocator)
			if _, exists := seenResources[resourceKey]; exists {
				return result, publication, fmt.Errorf(
					"%w: frontend %q emitted duplicate resource %q/%q",
					spec.ErrInvalidRequest,
					frontend.ID(),
					entry.Locator,
					decodedArtifact.SubresourceLocator,
				)
			}
			seenResources[resourceKey] = struct{}{}

			definition, err := baseutils.CanonicalizeDefinition(decodedArtifact.Definition)
			if err != nil {
				canonicalDiagnostics := []spec.Diagnostic{{
					Severity: spec.DiagnosticSeverityError,
					Code:     "artifactstore.definition.canonical.invalid",
					Message:  err.Error(),
				}}
				publication.Resources = append(
					publication.Resources,
					invalidCatalogResource(
						source.SourceID,
						entry.Locator,
						decodedArtifact.SubresourceLocator,
						frontend.ID(),
						now,
						&digest,
						canonicalDiagnostics,
					),
				)
				result.InvalidResources++
				result.Diagnostics = appendBoundedDiagnostics(
					result.Diagnostics,
					canonicalDiagnostics...,
				)
				continue
			}
			allDiagnostics := append([]spec.Diagnostic{}, diagnostics...)
			allDiagnostics = append(allDiagnostics, frontend.ValidateStructure(ctx, definition)...)
			allDiagnostics = append(allDiagnostics, frontend.ValidateSemantic(ctx, definition)...)
			selectors, dependencyDiagnostics := frontend.ExtractDependencies(ctx, definition)
			allDiagnostics = append(allDiagnostics, dependencyDiagnostics...)
			if err := validate.ValidateDiagnostics(allDiagnostics); err != nil {
				return result, publication, fmt.Errorf(
					"%w: frontend %q returned invalid diagnostics: %w",
					spec.ErrInvalidRequest,
					frontend.ID(),
					err,
				)
			}
			if err := errorDiagnostics("frontend validation", allDiagnostics); err != nil {
				publication.Resources = append(
					publication.Resources,
					invalidCatalogResource(
						source.SourceID,
						entry.Locator,
						decodedArtifact.SubresourceLocator,
						frontend.ID(),
						now,
						&digest,
						allDiagnostics,
					),
				)
				result.InvalidResources++
				result.Diagnostics = appendBoundedDiagnostics(
					result.Diagnostics,
					allDiagnostics...,
				)
				continue
			}

			definition.DependencySelectors = selectors
			definition.Digest = ""
			definition, err = baseutils.CanonicalizeDefinition(definition)
			if err != nil {
				dependencyDigestDiagnostics := []spec.Diagnostic{{
					Severity: spec.DiagnosticSeverityError,
					Code:     "artifactstore.definition.dependencies.invalid",
					Message:  err.Error(),
				}}
				publication.Resources = append(
					publication.Resources,
					invalidCatalogResource(
						source.SourceID,
						entry.Locator,
						decodedArtifact.SubresourceLocator,
						frontend.ID(),
						now,
						&digest,
						dependencyDigestDiagnostics,
					),
				)
				result.InvalidResources++
				result.Diagnostics = appendBoundedDiagnostics(
					result.Diagnostics,
					dependencyDigestDiagnostics...,
				)
				continue
			}
			if s.portableContent == nil {
				return result, publication, fmt.Errorf(
					"%w: portable content repository is not configured",
					spec.ErrUnsupported,
				)
			}
			stored, err := s.portableContent.PutDefinition(
				ctx,
				spec.ArtifactDefinitionFile{Format: spec.ArtifactDefinitionFileFormatV1, Definition: definition},
			)
			if err != nil {
				return result, publication, err
			}
			storedDigest := stored.Digest
			resource := spec.CatalogResource{
				SourceID:                source.SourceID,
				Locator:                 entry.Locator,
				SubresourceLocator:      decodedArtifact.SubresourceLocator,
				Kind:                    stored.Kind,
				LogicalName:             stored.LogicalName,
				LogicalVersion:          stored.LogicalVersion,
				CurrentDefinitionDigest: &storedDigest,
				SourceContentDigest:     &digest,
				FrontendID:              frontend.ID(),
				State:                   spec.CatalogStateValid,
				FirstSeenAt:             now,
				LastSeenAt:              now,
				Diagnostics:             allDiagnostics,
			}
			publication.Resources = append(publication.Resources, resource)
			publication.Revisions = append(
				publication.Revisions,
				spec.CatalogResourceRevision{
					SourceID:            source.SourceID,
					Locator:             entry.Locator,
					SubresourceLocator:  decodedArtifact.SubresourceLocator,
					DefinitionDigest:    storedDigest,
					SourceContentDigest: digest,
					Kind:                stored.Kind,
					FrontendID:          frontend.ID(),
					FirstSeenAt:         now,
					LastSeenAt:          now,
				},
			)
			result.ValidResources++
			result.Diagnostics = appendBoundedDiagnostics(
				result.Diagnostics,
				allDiagnostics...,
			)
		}

		// Once a candidate has decoded successfully, the emitted subresource set
		// is authoritative for that candidate. Otherwise removed MCP servers and
		// other multi-resource children remain valid forever after partial scans.
		for _, previous := range existingByLocator[entry.Locator] {
			if !frontendInScanScope(previous.FrontendID, plan.AllowedFrontendIDs) {
				continue
			}
			if _, emitted := seenResources[string(entry.Locator)+"\x00"+string(previous.SubresourceLocator)]; emitted {
				continue
			}
			previous.State = spec.CatalogStateMissing
			previous.Diagnostics = []spec.Diagnostic{}
			publication.Resources = append(publication.Resources, previous)
		}
	}

	if plan.Authoritative {
		published := make(map[string]struct{}, len(publication.Resources))
		for _, resource := range publication.Resources {
			published[recordOccurrenceKey(
				resource.SourceID,
				resource.Locator,
				resource.SubresourceLocator,
			)] = struct{}{}
		}
		for _, existing := range existingResources {
			if !catalogResourceInAuthoritativeScope(existing, plan) {
				continue
			}
			key := recordOccurrenceKey(existing.SourceID, existing.Locator, existing.SubresourceLocator)
			if _, ok := published[key]; ok {
				continue
			}
			existing.State = spec.CatalogStateMissing
			existing.Diagnostics = []spec.Diagnostic{}
			publication.Resources = append(publication.Resources, existing)
		}
	}

	beforeDigest, err := digestCatalog(existingResources)
	if err != nil {
		return result, publication, err
	}
	projected := make(map[string]spec.CatalogResource, len(existingResources))
	for _, existing := range existingResources {
		projected[recordOccurrenceKey(
			existing.SourceID,
			existing.Locator,
			existing.SubresourceLocator,
		)] = existing
	}
	applySourceCatalogPublication(projected, publication)
	afterDigest, err := digestCatalog(catalogResourcesFromMap(projected))
	if err != nil {
		return result, publication, err
	}
	publication.AdvanceObservationRevision = source.LastObservedGeneration == nil || beforeDigest != afterDigest
	if publication.AdvanceObservationRevision &&
		source.ObservationRevision >= spec.MaxObservationRevision {
		return result, publication, fmt.Errorf(
			"%w: source %q observation revision is exhausted",
			spec.ErrConflict,
			source.SourceID,
		)
	}
	publication.Diagnostics = append([]spec.Diagnostic(nil), result.Diagnostics...)
	return result, publication, nil
}

func (s *Store) selectFrontend(
	ctx context.Context,
	candidate spec.ArtifactCandidate,
	allowed []spec.FrontendID,
) (spec.ArtifactFrontend, error) {
	allowedSet := map[spec.FrontendID]struct{}{}
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	var selected spec.ArtifactFrontend
	best := spec.RecognitionNone
	tied := make([]spec.FrontendID, 0, 2)
	for _, frontend := range s.frontendsSnapshot() {
		if len(allowedSet) != 0 {
			if _, permitted := allowedSet[frontend.ID()]; !permitted {
				continue
			}
		}
		recognition := frontend.Recognizes(ctx, candidate)
		if recognition < spec.RecognitionNone || recognition > spec.RecognitionPreferred {
			return nil, fmt.Errorf(
				"%w: frontend %q returned invalid recognition %d",
				spec.ErrInvalidRequest,
				frontend.ID(),
				recognition,
			)
		}
		if recognition > best {
			selected, best = frontend, recognition
			tied = []spec.FrontendID{frontend.ID()}
		} else if recognition == best && recognition != spec.RecognitionNone {
			tied = append(tied, frontend.ID())
		}
	}
	if len(tied) > 1 {
		slices.Sort(tied)
		return nil, fmt.Errorf(
			"%w: candidate %q is equally recognized by frontends %v",
			spec.ErrConflict,
			candidate.Locator,
			tied,
		)
	}
	if selected == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}
	return selected, nil
}

func (s *Store) validateSourceScanPlan(plan spec.SourceScanPlan) error {
	if plan.SourceID == "" {
		return fmt.Errorf("%w: scan source ID is empty", spec.ErrInvalidRequest)
	}
	if plan.MaxFileBytes < 0 ||
		plan.MaxCandidates < 0 ||
		plan.MaxTraversalEntries < 0 ||
		plan.MaxTraversalDepth < 0 {
		return fmt.Errorf("%w: scan limits must not be negative", spec.ErrInvalidRequest)
	}
	if plan.Authoritative &&
		len(plan.ExplicitLocators) == 0 &&
		len(plan.DirectoryRoots) == 0 {
		return fmt.Errorf(
			"%w: an authoritative scan requires an explicit locator or directory scope",
			spec.ErrInvalidRequest,
		)
	}
	seenFrontends := make(map[spec.FrontendID]struct{}, len(plan.AllowedFrontendIDs))
	for _, id := range plan.AllowedFrontendIDs {
		if _, duplicate := seenFrontends[id]; duplicate {
			return fmt.Errorf("%w: duplicate allowed frontend %q", spec.ErrInvalidRequest, id)
		}
		seenFrontends[id] = struct{}{}
		if _, ok := s.frontendFor(id); !ok {
			return fmt.Errorf("%w: allowed frontend %q", spec.ErrFrontendUnavailable, id)
		}
	}
	seenLocators := make(map[spec.SourceLocator]struct{}, len(plan.ExplicitLocators))
	for _, locator := range plan.ExplicitLocators {
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf("%w: duplicate explicit locator %q", spec.ErrInvalidRequest, locator)
		}
		seenLocators[locator] = struct{}{}
	}
	return nil
}

func sourceLocatorInScanScope(locator spec.SourceLocator, plan spec.SourceScanPlan) bool {
	if slices.Contains(plan.ExplicitLocators, locator) {
		return true
	}
	for _, root := range plan.DirectoryRoots {
		walkRoot := root.Root
		if walkRoot == "" {
			walkRoot = "."
		}
		if matchesDirectoryRoot(walkRoot, locator, root) {
			return true
		}
	}
	return false
}

func frontendInScanScope(
	frontendID spec.FrontendID,
	allowed []spec.FrontendID,
) bool {
	if frontendID == "" || len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, frontendID)
}

func catalogResourcesForFrontendScope(
	resources []spec.CatalogResource,
	allowed []spec.FrontendID,
) []spec.CatalogResource {
	out := make([]spec.CatalogResource, 0, len(resources))
	for _, resource := range resources {
		if frontendInScanScope(resource.FrontendID, allowed) {
			out = append(out, resource)
		}
	}
	return out
}

func catalogResourceInAuthoritativeScope(
	resource spec.CatalogResource,
	plan spec.SourceScanPlan,
) bool {
	if !sourceLocatorInScanScope(resource.Locator, plan) {
		return false
	}
	return frontendInScanScope(resource.FrontendID, plan.AllowedFrontendIDs)
}

func collectSourceCandidates(
	ctx context.Context,
	driver spec.SourceDriver,
	source spec.ArtifactSource,
	plan spec.SourceScanPlan,
) ([]spec.SourceEntry, error) {
	maxCandidates := plan.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = spec.DefaultMaxScanCandidates
	}
	maxEntries := plan.MaxTraversalEntries
	if maxEntries <= 0 {
		maxEntries = spec.DefaultMaxScanEntries
	}
	maxDepth := plan.MaxTraversalDepth
	if maxDepth <= 0 {
		maxDepth = spec.DefaultMaxTraversalDepth
	}
	seen := map[spec.SourceLocator]spec.SourceEntry{}
	add := func(entry spec.SourceEntry) error {
		if _, exists := seen[entry.Locator]; !exists && len(seen) >= maxCandidates {
			return fmt.Errorf("%w: scan exceeds %d candidates", spec.ErrInvalidRequest, maxCandidates)
		}
		seen[entry.Locator] = entry
		return nil
	}
	for _, locator := range plan.ExplicitLocators {
		entry, err := driver.Stat(ctx, source, locator)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		if entry.IsRegular {
			if err := add(entry); err != nil {
				return nil, err
			}
		}
	}
	visitedEntries := 0
	for _, root := range plan.DirectoryRoots {
		walkRoot := root.Root
		if walkRoot == "" {
			walkRoot = "."
		}
		for _, pattern := range root.IncludePatterns {
			if _, err := path.Match(pattern, "candidate"); err != nil {
				return nil, fmt.Errorf(
					"%w: invalid include pattern %q: %w",
					spec.ErrInvalidRequest,
					pattern,
					err,
				)
			}
		}
		rootEntry, err := driver.Stat(ctx, source, walkRoot)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		if !rootEntry.IsDirectory {
			return nil, fmt.Errorf(
				"%w: scan root %q is not a directory",
				spec.ErrInvalidRequest,
				walkRoot,
			)
		}
		var visit func(spec.SourceLocator, int) error
		visit = func(directory spec.SourceLocator, depth int) error {
			entries, err := driver.ReadDir(ctx, source, directory)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return err
				}
				visitedEntries++
				if visitedEntries > maxEntries {
					return fmt.Errorf(
						"%w: scan traversal exceeds %d entries",
						spec.ErrInvalidRequest,
						maxEntries,
					)
				}
				entryDepth := depth + 1
				if entryDepth > maxDepth {
					return fmt.Errorf(
						"%w: scan traversal exceeds depth %d at %q",
						spec.ErrInvalidRequest,
						maxDepth,
						entry.Locator,
					)
				}
				if entry.IsDirectory {
					if root.Recursive {
						if err := visit(entry.Locator, entryDepth); err != nil {
							return err
						}
					}
					continue
				}
				if entry.IsRegular && matchesDirectoryRoot(walkRoot, entry.Locator, root) {
					if err := add(entry); err != nil {
						return err
					}
				}
			}
			return nil
		}
		if err := visit(walkRoot, 0); err != nil {
			return nil, err
		}
	}
	out := make([]spec.SourceEntry, 0, len(seen))
	for _, entry := range seen {
		out = append(out, entry)
	}
	sort.Slice(out, func(left, right int) bool { return out[left].Locator < out[right].Locator })
	return out, nil
}

func matchesDirectoryRoot(root, locator spec.SourceLocator, plan spec.DirectoryScanRoot) bool {
	base := string(root)
	value := string(locator)
	relative := value
	if base == "" {
		base = "."
	}
	if base != "." {
		prefix := base + "/"
		if !strings.HasPrefix(value, prefix) {
			return false
		}
		relative = strings.TrimPrefix(value, prefix)
	}
	if !plan.Recursive && strings.Contains(relative, "/") {
		return false
	}
	if len(plan.IncludePatterns) == 0 {
		return true
	}
	for _, pattern := range plan.IncludePatterns {
		if matched, _ := path.Match(pattern, relative); matched {
			return true
		}
	}
	return false
}

func readCandidate(
	ctx context.Context,
	driver spec.SourceDriver,
	source spec.ArtifactSource,
	entry spec.SourceEntry,
	maximum int64,
) ([]byte, error) {
	if maximum <= 0 {
		maximum = spec.MaxDefinitionJSONBytes
	}
	if entry.SizeBytes > maximum {
		return nil, fmt.Errorf(
			"%w: candidate %q exceeds %d bytes",
			spec.ErrInvalidRequest,
			entry.Locator,
			maximum,
		)
	}
	reader, err := driver.Open(ctx, source, entry.Locator)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		_ = reader.Close()
		return nil, err
	}
	if err := reader.Close(); err != nil {
		return nil, err
	}
	if int64(len(content)) > maximum {
		return nil, fmt.Errorf(
			"%w: candidate %q exceeds %d bytes",
			spec.ErrInvalidRequest,
			entry.Locator,
			maximum,
		)
	}
	return content, nil
}

func invalidCatalogResource(
	sourceID spec.SourceID,
	locator spec.SourceLocator,
	subresource spec.SubresourceLocator,
	frontendID spec.FrontendID,
	now time.Time,
	sourceContentDigest *spec.Digest,
	diagnostics []spec.Diagnostic,
) spec.CatalogResource {
	return spec.CatalogResource{
		SourceID:            sourceID,
		Locator:             locator,
		SubresourceLocator:  subresource,
		SourceContentDigest: sourceContentDigest,
		FrontendID:          frontendID,
		State:               spec.CatalogStateInvalid,
		FirstSeenAt:         now,
		LastSeenAt:          now,
		Diagnostics:         diagnostics,
	}
}

func invalidCandidateResources(
	sourceID spec.SourceID,
	locator spec.SourceLocator,
	frontendID spec.FrontendID,
	now time.Time,
	sourceContentDigest *spec.Digest,
	diagnostics []spec.Diagnostic,
	existing []spec.CatalogResource,
) []spec.CatalogResource {
	if len(existing) == 0 {
		return []spec.CatalogResource{invalidCatalogResource(
			sourceID,
			locator,
			"",
			frontendID,
			now,
			sourceContentDigest,
			diagnostics,
		)}
	}
	out := make([]spec.CatalogResource, 0, len(existing))
	for _, previous := range existing {
		resource := previous
		resource.State = spec.CatalogStateInvalid
		resource.LastSeenAt = now
		resource.Diagnostics = append([]spec.Diagnostic(nil), diagnostics...)
		resource.SourceContentDigest = cloneDigest(sourceContentDigest)
		if frontendID != "" {
			resource.FrontendID = frontendID
		}
		out = append(out, resource)
	}
	return out
}

func digestScanPlan(plan spec.ScanPlan) (spec.Digest, error) {
	normalized := spec.ScanPlan{
		SourcePlans: append([]spec.SourceScanPlan(nil), plan.SourcePlans...),
	}
	for index := range normalized.SourcePlans {
		sourcePlan := &normalized.SourcePlans[index]
		sourcePlan.ExplicitLocators = append([]spec.SourceLocator(nil), sourcePlan.ExplicitLocators...)
		sourcePlan.DirectoryRoots = append([]spec.DirectoryScanRoot(nil), sourcePlan.DirectoryRoots...)
		sourcePlan.AllowedFrontendIDs = append([]spec.FrontendID(nil), sourcePlan.AllowedFrontendIDs...)
		slices.Sort(sourcePlan.ExplicitLocators)
		slices.Sort(sourcePlan.AllowedFrontendIDs)
		for rootIndex := range sourcePlan.DirectoryRoots {
			sourcePlan.DirectoryRoots[rootIndex].IncludePatterns = append(
				[]string(nil),
				sourcePlan.DirectoryRoots[rootIndex].IncludePatterns...,
			)
			sort.Strings(sourcePlan.DirectoryRoots[rootIndex].IncludePatterns)
		}
		sort.Slice(sourcePlan.DirectoryRoots, func(left, right int) bool {
			return sourcePlan.DirectoryRoots[left].Root < sourcePlan.DirectoryRoots[right].Root
		})
	}
	sort.Slice(normalized.SourcePlans, func(left, right int) bool {
		return normalized.SourcePlans[left].SourceID < normalized.SourcePlans[right].SourceID
	})

	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return "", err
	}
	return baseutils.DigestBytes(canonical), nil
}

func digestCatalog(resources []spec.CatalogResource) (spec.Digest, error) {
	type digestResource struct {
		SourceID                spec.SourceID           `json:"sourceID"`
		Locator                 spec.SourceLocator      `json:"locator"`
		SubresourceLocator      spec.SubresourceLocator `json:"subresourceLocator,omitempty"`
		PackageManifestLocator  spec.SourceLocator      `json:"packageManifestLocator,omitempty"`
		Kind                    spec.ArtifactKind       `json:"kind,omitempty"`
		LogicalName             spec.LogicalName        `json:"logicalName,omitempty"`
		LogicalVersion          spec.LogicalVersion     `json:"logicalVersion,omitempty"`
		CurrentDefinitionDigest *spec.Digest            `json:"currentDefinitionDigest,omitempty"`
		SourceContentDigest     *spec.Digest            `json:"sourceContentDigest,omitempty"`
		FrontendID              spec.FrontendID         `json:"frontendID,omitempty"`
		State                   spec.CatalogState       `json:"state"`
		Diagnostics             []spec.Diagnostic       `json:"diagnostics,omitempty"`
	}
	projected := make([]digestResource, 0, len(resources))
	for _, resource := range resources {
		projected = append(projected, digestResource{
			SourceID:                resource.SourceID,
			Locator:                 resource.Locator,
			SubresourceLocator:      resource.SubresourceLocator,
			PackageManifestLocator:  resource.PackageManifestLocator,
			Kind:                    resource.Kind,
			LogicalName:             resource.LogicalName,
			LogicalVersion:          resource.LogicalVersion,
			CurrentDefinitionDigest: resource.CurrentDefinitionDigest,
			SourceContentDigest:     resource.SourceContentDigest,
			FrontendID:              resource.FrontendID,
			State:                   resource.State,
			Diagnostics:             resource.Diagnostics,
		})
	}
	sort.Slice(projected, func(left, right int) bool {
		if projected[left].SourceID != projected[right].SourceID {
			return projected[left].SourceID < projected[right].SourceID
		}
		if projected[left].Locator != projected[right].Locator {
			return projected[left].Locator < projected[right].Locator
		}
		return projected[left].SubresourceLocator < projected[right].SubresourceLocator
	})
	raw, err := json.Marshal(projected)
	if err != nil {
		return "", err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return "", err
	}
	return baseutils.DigestBytes(canonical), nil
}

func applySourceCatalogPublication(
	resources map[string]spec.CatalogResource,
	publication spec.SourceCatalogPublication,
) {
	for _, incoming := range publication.Resources {
		key := recordOccurrenceKey(
			incoming.SourceID,
			incoming.Locator,
			incoming.SubresourceLocator,
		)
		if existing, ok := resources[key]; ok {
			incoming.FirstSeenAt = existing.FirstSeenAt
		}
		resources[key] = incoming
	}
}

func catalogResourcesFromMap(
	resources map[string]spec.CatalogResource,
) []spec.CatalogResource {
	out := make([]spec.CatalogResource, 0, len(resources))
	for _, resource := range resources {
		out = append(out, resource)
	}
	sort.Slice(out, func(left, right int) bool {
		if out[left].SourceID != out[right].SourceID {
			return out[left].SourceID < out[right].SourceID
		}
		if out[left].Locator != out[right].Locator {
			return out[left].Locator < out[right].Locator
		}
		return out[left].SubresourceLocator < out[right].SubresourceLocator
	})
	return out
}
