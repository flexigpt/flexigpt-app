package artifactstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Store) ScanRoot(ctx context.Context, rootID spec.RootID, plan spec.ScanPlan) (spec.ScanResult, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ScanResult{}, err
	}
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return spec.ScanResult{}, err
	}
	attachments, err := s.repository.ListRootSourceAttachments(ctx, rootID)
	if err != nil {
		return spec.ScanResult{}, err
	}
	plans := make(map[spec.SourceID]spec.SourceScanPlan)
	for _, sourcePlan := range plan.SourcePlans {
		if sourcePlan.SourceID != "" {
			plans[sourcePlan.SourceID] = sourcePlan
		}
	}
	result := spec.ScanResult{RootID: rootID}
	sourceGenerations := map[spec.SourceID]spec.SourceGeneration{}
	for _, attachment := range attachments {
		if !attachment.Enabled {
			continue
		}
		source, err := s.repository.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return spec.ScanResult{}, err
		}
		if !source.Enabled {
			continue
		}
		sourcePlan, ok := plans[source.SourceID]
		if !ok {
			sourcePlan = spec.SourceScanPlan{
				SourceID:       source.SourceID,
				DirectoryRoots: []spec.DirectoryScanRoot{{Root: ".", Recursive: true}},
				MaxFileBytes:   spec.MaxDefinitionJSONBytes,
				Authoritative:  true,
			}
		}
		sourceResult, publication, err := s.scanSource(ctx, source, sourcePlan)
		if err != nil {
			return spec.ScanResult{}, err
		}
		if err := s.repository.PublishSourceCatalog(ctx, publication); err != nil {
			return spec.ScanResult{}, err
		}
		result.Sources = append(result.Sources, sourceResult)
		sourceGenerations[source.SourceID] = sourceResult.Generation
		result.Diagnostics = append(result.Diagnostics, sourceResult.Diagnostics...)
	}
	resources, err := s.repository.ListCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return spec.ScanResult{}, err
	}
	planDigest, err := digestScanPlan(plan)
	if err != nil {
		return spec.ScanResult{}, err
	}
	catalogDigest, err := digestCatalog(resources)
	if err != nil {
		return spec.ScanResult{}, err
	}
	generation, err := s.repository.PublishRootCatalogGeneration(
		ctx,
		spec.RootCatalogPublication{
			RootID:            rootID,
			SourceGenerations: sourceGenerations,
			ScanPlanDigest:    planDigest,
			CatalogDigest:     catalogDigest,
			CreatedAt:         s.nowUTC(),
			Diagnostics:       result.Diagnostics,
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
		SourceID:           source.SourceID,
		ObservedGeneration: generation,
		ObservedAt:         now,
		Authoritative:      plan.Authoritative,
	}
	result := spec.SourceScanResult{SourceID: source.SourceID, Generation: generation}
	for _, entry := range entries {
		result.Candidates++
		content, err := readCandidate(ctx, driver, source, entry.Locator, plan.MaxFileBytes)
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
		frontend := s.selectFrontend(ctx, candidate, plan.AllowedFrontendIDs)
		if frontend == nil {
			continue
		}
		decoded, diagnostics := frontend.Decode(ctx, candidate)
		if err := errorDiagnostics("frontend decode", diagnostics); err != nil || len(decoded) == 0 {
			publication.Resources = append(
				publication.Resources,
				invalidCatalogResource(source.SourceID, entry.Locator, "", frontend.ID(), now, diagnostics),
			)
			result.InvalidResources++
			result.Diagnostics = append(result.Diagnostics, diagnostics...)
			continue
		}
		for _, decodedArtifact := range decoded {
			definition, err := baseutils.CanonicalizeDefinition(decodedArtifact.Definition)
			if err != nil {
				publication.Resources = append(
					publication.Resources,
					invalidCatalogResource(
						source.SourceID,
						entry.Locator,
						decodedArtifact.SubresourceLocator,
						frontend.ID(),
						now,
						[]spec.Diagnostic{
							{
								Severity: spec.DiagnosticSeverityError,
								Code:     "artifactstore.definition.canonical.invalid",
								Message:  err.Error(),
							},
						},
					),
				)
				result.InvalidResources++
				continue
			}
			allDiagnostics := append([]spec.Diagnostic{}, frontend.ValidateStructure(ctx, definition)...)
			allDiagnostics = append(allDiagnostics, frontend.ValidateSemantic(ctx, definition)...)
			selectors, dependencyDiagnostics := frontend.ExtractDependencies(ctx, definition)
			allDiagnostics = append(allDiagnostics, dependencyDiagnostics...)
			if err := errorDiagnostics("frontend validation", allDiagnostics); err != nil {
				publication.Resources = append(
					publication.Resources,
					invalidCatalogResource(
						source.SourceID,
						entry.Locator,
						decodedArtifact.SubresourceLocator,
						frontend.ID(),
						now,
						allDiagnostics,
					),
				)
				result.InvalidResources++
				result.Diagnostics = append(result.Diagnostics, allDiagnostics...)
				continue
			}
			definition.DependencySelectors = selectors
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
	return result, publication, nil
}

func (s *Store) selectFrontend(
	ctx context.Context,
	candidate spec.ArtifactCandidate,
	allowed []spec.FrontendID,
) spec.ArtifactFrontend {
	allowedSet := map[spec.FrontendID]struct{}{}
	for _, id := range allowed {
		allowedSet[id] = struct{}{}
	}
	var selected spec.ArtifactFrontend
	best := spec.RecognitionNone
	for _, frontend := range s.frontendsSnapshot() {
		if len(allowedSet) > 0 {
			if _, ok := allowedSet[frontend.ID()]; !ok {
				continue
			}
		}
		recognition := frontend.Recognizes(ctx, candidate)
		if recognition > best {
			selected, best = frontend, recognition
		}
	}
	return selected
}

func collectSourceCandidates(
	ctx context.Context,
	driver spec.SourceDriver,
	source spec.ArtifactSource,
	plan spec.SourceScanPlan,
) ([]spec.SourceEntry, error) {
	seen := map[spec.SourceLocator]spec.SourceEntry{}
	for _, locator := range plan.ExplicitLocators {
		entry, err := driver.Stat(ctx, source, locator)
		if err != nil {
			return nil, err
		}
		if entry.IsRegular {
			seen[entry.Locator] = entry
		}
	}
	for _, root := range plan.DirectoryRoots {
		walkRoot := root.Root
		if walkRoot == "" {
			walkRoot = "."
		}
		err := driver.Walk(ctx, source, walkRoot, func(ctx context.Context, entry spec.SourceEntry) error {
			if !entry.IsRegular || !matchesDirectoryRoot(walkRoot, entry.Locator, root) {
				return nil
			}
			seen[entry.Locator] = entry
			return nil
		})
		if err != nil {
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
	locator spec.SourceLocator,
	maximum int64,
) ([]byte, error) {
	if maximum <= 0 {
		maximum = spec.MaxDefinitionJSONBytes
	}
	reader, err := driver.Open(ctx, source, locator)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maximum {
		return nil, fmt.Errorf("%w: candidate %q exceeds %d bytes", spec.ErrInvalidRequest, locator, maximum)
	}
	return content, nil
}

func invalidCatalogResource(
	sourceID spec.SourceID,
	locator spec.SourceLocator,
	subresource spec.SubresourceLocator,
	frontendID spec.FrontendID,
	now time.Time,
	diagnostics []spec.Diagnostic,
) spec.CatalogResource {
	return spec.CatalogResource{
		SourceID:           sourceID,
		Locator:            locator,
		SubresourceLocator: subresource,
		FrontendID:         frontendID,
		State:              spec.CatalogStateInvalid,
		FirstSeenAt:        now,
		LastSeenAt:         now,
		Diagnostics:        diagnostics,
	}
}

func digestScanPlan(plan spec.ScanPlan) (spec.Digest, error) {
	raw, err := json.Marshal(plan)
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
	raw, err := json.Marshal(resources)
	if err != nil {
		return "", err
	}
	canonical, err := baseutils.CanonicalizeJSON(raw)
	if err != nil {
		return "", err
	}
	return baseutils.DigestBytes(canonical), nil
}
