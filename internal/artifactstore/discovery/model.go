package discovery

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
)

const (
	DiagnosticCodeCandidateTooLarge         = "artifact.discovery.candidate-too-large"
	DiagnosticCodeDecoderAmbiguous          = "artifact.discovery.decoder-ambiguous"
	DiagnosticCodeDecoderInvalidRecognition = "artifact.discovery.decoder-invalid-recognition"
	DiagnosticCodeDefinitionInvalid         = "artifact.discovery.definition-invalid"
	DiagnosticCodeResourceMissing           = "artifact.discovery.resource-missing"
	DiagnosticCodeSubresourceMissing        = "artifact.discovery.subresource-missing"
)

type DirectoryRoot struct {
	Root            artifactstore.Locator
	Recursive       bool
	IncludePatterns []string
}

type SourcePlan struct {
	SourceID           artifactstore.SourceID
	ExplicitLocators   []artifactstore.Locator
	DirectoryRoots     []DirectoryRoot
	ExpectedGeneration string
	AllowedDecoderIDs  []artifactstore.DecoderID
	Authoritative      bool
	MaxCandidateBytes  int64
	MaxTotalBytes      int64
	MaxCandidates      int
	MaxEntries         int
	MaxDepth           int
}

func (p *SourcePlan) Validate() error {
	if err := artifactstore.ValidateSourceID(p.SourceID); err != nil {
		return err
	}
	if len(p.ExplicitLocators) == 0 &&
		len(p.DirectoryRoots) == 0 {
		return fmt.Errorf(
			"%w: source discovery plan has no scope",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidateOptionalText(
		"expected source generation",
		p.ExpectedGeneration,
		artifactstore.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if p.MaxCandidateBytes < 0 ||
		p.MaxTotalBytes < 0 ||
		p.MaxCandidates < 0 ||
		p.MaxEntries < 0 ||
		p.MaxDepth < 0 {
		return fmt.Errorf(
			"%w: discovery limits cannot be negative",
			artifactstore.ErrInvalid,
		)
	}
	seenLocators := make(map[artifactstore.Locator]struct{}, len(p.ExplicitLocators))
	for _, locator := range p.ExplicitLocators {
		if err := artifactstore.ValidateLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate explicit discovery locator %q",
				artifactstore.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenRoots := make(map[artifactstore.Locator]struct{}, len(p.DirectoryRoots))
	for _, root := range p.DirectoryRoots {
		if err := artifactstore.ValidateLocator(root.Root, true); err != nil {
			return err
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery root %q",
				artifactstore.ErrInvalid,
				root.Root,
			)
		}
		seenRoots[root.Root] = struct{}{}

		seenPatterns := make(map[string]struct{}, len(root.IncludePatterns))
		for _, pattern := range root.IncludePatterns {
			if strings.TrimSpace(pattern) != pattern || pattern == "" {
				return fmt.Errorf(
					"%w: discovery pattern must be non-empty and trimmed",
					artifactstore.ErrInvalid,
				)
			}
			if _, duplicate := seenPatterns[pattern]; duplicate {
				return fmt.Errorf(
					"%w: duplicate discovery pattern %q",
					artifactstore.ErrInvalid,
					pattern,
				)
			}
			seenPatterns[pattern] = struct{}{}
			if _, err := path.Match(pattern, "candidate"); err != nil {
				return fmt.Errorf(
					"%w: invalid discovery pattern %q: %w",
					artifactstore.ErrInvalid,
					pattern,
					err,
				)
			}
		}
	}

	seenDecoders := make(map[artifactstore.DecoderID]struct{}, len(p.AllowedDecoderIDs))
	for _, decoderID := range p.AllowedDecoderIDs {
		if err := artifactstore.ValidateDecoderID(decoderID); err != nil {
			return err
		}
		if _, duplicate := seenDecoders[decoderID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate allowed decoder %q",
				artifactstore.ErrInvalid,
				decoderID,
			)
		}
		seenDecoders[decoderID] = struct{}{}
	}
	return nil
}

func (p *SourcePlan) ApplyDefaults() {
	if p.MaxCandidateBytes <= 0 {
		p.MaxCandidateBytes = artifactstore.MaxCandidateBytes
	}
	if p.MaxTotalBytes <= 0 {
		p.MaxTotalBytes = artifactstore.MaxScanBytes
	}
	if p.MaxCandidates <= 0 {
		p.MaxCandidates = artifactstore.DefaultMaxCandidates
	}
	if p.MaxEntries <= 0 {
		p.MaxEntries = artifactstore.DefaultMaxEntries
	}
	if p.MaxDepth <= 0 {
		p.MaxDepth = artifactstore.DefaultMaxDepth
	}
	slices.Sort(p.ExplicitLocators)
	sort.Slice(p.DirectoryRoots, func(left, right int) bool {
		return p.DirectoryRoots[left].Root < p.DirectoryRoots[right].Root
	})
	slices.Sort(p.AllowedDecoderIDs)
}

type Plan struct {
	Sources []SourcePlan
}

func (p Plan) Validate() error {
	seen := make(map[artifactstore.SourceID]struct{}, len(p.Sources))
	for index, sourcePlan := range p.Sources {
		if err := sourcePlan.Validate(); err != nil {
			return fmt.Errorf("source plan %d: %w", index, err)
		}
		if _, duplicate := seen[sourcePlan.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate source plan for %q",
				artifactstore.ErrInvalid,
				sourcePlan.SourceID,
			)
		}
		seen[sourcePlan.SourceID] = struct{}{}
	}
	return nil
}

func (p Plan) BySource() map[artifactstore.SourceID]SourcePlan {
	output := make(map[artifactstore.SourceID]SourcePlan, len(p.Sources))
	for _, value := range p.Sources {
		value.ApplyDefaults()
		output[value.SourceID] = value
	}
	return output
}

type Result struct {
	Occurrences []catalog.Occurrence
	Definitions map[artifactstore.Digest]definition.Definition
	Diagnostics []artifactstore.Diagnostic
	Candidates  int
}
