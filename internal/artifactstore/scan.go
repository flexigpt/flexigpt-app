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
	attachmentExpectations := make(
		[]spec.RootScanAttachmentExpectation,
		0,
		len(attachments),
	)
	sourceExpectations := make([]spec.RootScanSourceExpectation, 0, len(attachments))
	catalogByOccurrence := make(map[string]spec.CatalogResource)
	sourceGenerations := map[spec.SourceID]spec.SourceGeneration{}
	for _, attachment := range attachments {
		attached[attachment.SourceID] = attachment
		attachmentExpectations = append(attachmentExpectations, spec.RootScanAttachmentExpectation{
			SourceID:   attachment.SourceID,
			ModifiedAt: attachment.ModifiedAt,
			Enabled:    attachment.Enabled,
		})
		source, err := s.repository.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return spec.ScanResult{}, err
		}
		sources[source.SourceID] = source
		sourceExpectations = append(sourceExpectations, spec.RootScanSourceExpectation{
			SourceID:            source.SourceID,
			ModifiedAt:          source.ModifiedAt,
			ObservationRevision: source.ObservationRevision,
			Enabled:             source.Enabled,
		})
		if !attachment.Enabled || !source.Enabled {
			continue
		}
		if source.LastObservedGeneration != nil {
			sourceGenerations[source.SourceID] = *source.LastObservedGeneration
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
		executedPlan.SourcePlans = append(executedPlan.SourcePlans, sourcePlan)
		sourceResult, publication, err := s.scanSource(ctx, source, sourcePlan)
		if err != nil {
			return spec.ScanResult{}, err
		}
		sourcePublications = append(sourcePublications, publication)
		applySourceCatalogPublication(catalogByOccurrence, publication)
		result.Sources = append(result.Sources, sourceResult)
		sourceGenerations[source.SourceID] = sourceResult.Generation
		result.Diagnostics = append(result.Diagnostics, sourceResult.Diagnostics...)
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
				RootID:            rootID,
				SourceGenerations: sourceGenerations,
				ScanPlanDigest:    planDigest,
				CatalogDigest:     catalogDigest,
				CreatedAt:         s.nowUTC(),
				Diagnostics:       result.Diagnostics,
			},
			ExpectedRootModifiedAt: root.ModifiedAt,
			Attachments:            attachmentExpectations,
			Sources:                sourceExpectations,
			SourceCatalogs:         sourcePublications,
			CatalogResources:       resources,
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
	generation, err := driver.Snapshot(ctx, source)
	if err != nil {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, err
	}
	entries, err := collectSourceCandidates(ctx, driver, source, plan)
	if err != nil {
		return spec.SourceScanResult{}, spec.SourceCatalogPublication{}, err
	}
	now := s.nowUTC()
	publication := spec.SourceCatalogPublication{
		SourceID:                    source.SourceID,
		ExpectedSourceModifiedAt:    source.ModifiedAt,
		ExpectedObservationRevision: source.ObservationRevision,
		ObservedGeneration:          generation,
		ObservedAt:                  now,
		Authoritative:               plan.Authoritative,
	}
	result := spec.SourceScanResult{SourceID: source.SourceID, Generation: generation}
	seenResources := make(map[string]struct{})
	for _, entry := range entries {
		result.Candidates++
		content, err := readCandidate(ctx, driver, source, entry, plan.MaxFileBytes)
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
		frontend, err := s.selectFrontend(ctx, candidate, plan.AllowedFrontendIDs)
		if err != nil {
			return result, publication, err
		}
		if frontend == nil {
			continue
		}
		decoded, diagnostics := frontend.Decode(ctx, candidate)
		if len(decoded) == 0 && len(diagnostics) == 0 {
			diagnostics = []spec.Diagnostic{{
				Severity: spec.DiagnosticSeverityError,
				Code:     "artifactstore.frontend.decode-empty",
				Message:  "the selected frontend emitted no definitions",
			}}
		}
		if err := spec.ValidateDiagnostics(diagnostics); err != nil {
			return result, publication, fmt.Errorf(
				"%w: frontend %q returned invalid diagnostics: %w",
				spec.ErrInvalidRequest,
				frontend.ID(),
				err,
			)
		}
		if err := errorDiagnostics("frontend decode", diagnostics); err != nil || len(decoded) == 0 {
			publication.Resources = append(
				publication.Resources,
				invalidCatalogResource(
					source.SourceID,
					entry.Locator,
					"",
					frontend.ID(),
					now,
					&digest,
					diagnostics,
				),
			)
			result.InvalidResources++
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
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
				result.Diagnostics = append(result.Diagnostics, canonicalDiagnostics...)
				continue
			}
			allDiagnostics := append([]spec.Diagnostic{}, diagnostics...)
			allDiagnostics = append(allDiagnostics, frontend.ValidateStructure(ctx, definition)...)
			allDiagnostics = append(allDiagnostics, frontend.ValidateSemantic(ctx, definition)...)
			selectors, dependencyDiagnostics := frontend.ExtractDependencies(ctx, definition)
			allDiagnostics = append(allDiagnostics, dependencyDiagnostics...)
			if err := spec.ValidateDiagnostics(allDiagnostics); err != nil {
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
				result.Diagnostics = append(result.Diagnostics, allDiagnostics...)
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
				result.Diagnostics = append(result.Diagnostics, dependencyDigestDiagnostics...)
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
		}
	}

	confirmedGeneration, err := driver.Snapshot(ctx, source)
	if err != nil {
		return result, publication, err
	}
	if confirmedGeneration != generation {
		return result, publication, fmt.Errorf(
			"%w: source %q changed while it was being scanned",
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
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[frontend.ID()]; !ok {
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
	return selected, nil
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
	seen := make(map[string]struct{}, len(publication.Resources))
	for _, incoming := range publication.Resources {
		key := recordOccurrenceKey(
			incoming.SourceID,
			incoming.Locator,
			incoming.SubresourceLocator,
		)
		seen[key] = struct{}{}
		if existing, ok := resources[key]; ok {
			incoming.FirstSeenAt = existing.FirstSeenAt
			if incoming.PackageManifestLocator == "" {
				incoming.PackageManifestLocator = existing.PackageManifestLocator
			}
			if incoming.Kind == "" {
				incoming.Kind = existing.Kind
			}
			if incoming.LogicalName == "" {
				incoming.LogicalName = existing.LogicalName
			}
			if incoming.LogicalVersion == "" {
				incoming.LogicalVersion = existing.LogicalVersion
			}
			if incoming.CurrentDefinitionDigest == nil {
				incoming.CurrentDefinitionDigest = cloneDigest(existing.CurrentDefinitionDigest)
			}
			if incoming.SourceContentDigest == nil {
				incoming.SourceContentDigest = cloneDigest(existing.SourceContentDigest)
			}
			if incoming.FrontendID == "" {
				incoming.FrontendID = existing.FrontendID
			}
		}
		resources[key] = incoming
	}
	if !publication.Authoritative {
		return
	}
	for key, resource := range resources {
		if resource.SourceID != publication.SourceID {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		resource.State = spec.CatalogStateMissing
		resource.Diagnostics = []spec.Diagnostic{}
		resources[key] = resource
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
