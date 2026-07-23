package discovery

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type DecoderHint struct {
	Locator    artifactstore.Locator
	Recursive  bool
	DecoderIDs []artifactstore.DecoderID
}

type SourcePlan struct {
	SourceID         artifactstore.SourceID
	ExplicitLocators []artifactstore.Locator
	DirectoryRoots   []DirectoryRoot
	DecoderHints     []DecoderHint

	ExpectedGeneration string
	AllowedDecoderIDs  []artifactstore.DecoderID
	Authoritative      bool
	MaxCandidateBytes  int64
	MaxTotalBytes      int64
	MaxCandidates      int
	MaxEntries         int
	MaxDepth           int
}

func (p SourcePlan) Validate() error {
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
	if p.ExpectedGeneration != "" {
		if err := artifactstore.ValidateSourceGeneration(
			p.ExpectedGeneration,
		); err != nil {
			return err
		}
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
	for index, hint := range p.DecoderHints {
		if err := artifactstore.ValidateLocator(hint.Locator, true); err != nil {
			return fmt.Errorf("decoder hint %d: %w", index, err)
		}
		if len(hint.DecoderIDs) == 0 {
			return fmt.Errorf(
				"%w: decoder hint %d has no decoder IDs",
				artifactstore.ErrInvalid,
				index,
			)
		}
		seenHintDecoders := make(map[artifactstore.DecoderID]struct{}, len(hint.DecoderIDs))
		for _, decoderID := range hint.DecoderIDs {
			if err := artifactstore.ValidateDecoderID(decoderID); err != nil {
				return err
			}
			if _, duplicate := seenHintDecoders[decoderID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate decoder hint ID %q",
					artifactstore.ErrInvalid,
					decoderID,
				)
			}
			seenHintDecoders[decoderID] = struct{}{}
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

// Normalized returns an owned, deterministic copy with default limits.
//
// It intentionally does not mutate the input plan or its backing slices.
func (p SourcePlan) Normalized() SourcePlan {
	output := p
	output.ExplicitLocators = append(
		[]artifactstore.Locator(nil),
		p.ExplicitLocators...,
	)
	output.AllowedDecoderIDs = append(
		[]artifactstore.DecoderID(nil),
		p.AllowedDecoderIDs...,
	)
	output.DecoderHints = make([]DecoderHint, len(p.DecoderHints))
	for index, hint := range p.DecoderHints {
		output.DecoderHints[index] = hint
		output.DecoderHints[index].DecoderIDs = append(
			[]artifactstore.DecoderID(nil),
			hint.DecoderIDs...,
		)
	}
	output.DirectoryRoots = make([]DirectoryRoot, len(p.DirectoryRoots))
	for index, root := range p.DirectoryRoots {
		output.DirectoryRoots[index] = root
		output.DirectoryRoots[index].IncludePatterns = append(
			[]string(nil),
			root.IncludePatterns...,
		)
	}

	if output.MaxCandidateBytes <= 0 {
		output.MaxCandidateBytes = artifactstore.MaxCandidateBytes
	}
	if output.MaxTotalBytes <= 0 {
		output.MaxTotalBytes = artifactstore.MaxScanBytes
	}
	if output.MaxCandidates <= 0 {
		output.MaxCandidates = artifactstore.DefaultMaxCandidates
	}
	if output.MaxEntries <= 0 {
		output.MaxEntries = artifactstore.DefaultMaxEntries
	}
	if output.MaxDepth <= 0 {
		output.MaxDepth = artifactstore.DefaultMaxDepth
	}
	slices.Sort(output.ExplicitLocators)
	sort.Slice(output.DirectoryRoots, func(left, right int) bool {
		return output.DirectoryRoots[left].Root < output.DirectoryRoots[right].Root
	})
	slices.Sort(output.AllowedDecoderIDs)
	for index := range output.DecoderHints {
		slices.Sort(output.DecoderHints[index].DecoderIDs)
	}
	sort.Slice(output.DecoderHints, func(left, right int) bool {
		if output.DecoderHints[left].Locator != output.DecoderHints[right].Locator {
			return output.DecoderHints[left].Locator < output.DecoderHints[right].Locator
		}
		if output.DecoderHints[left].Recursive != output.DecoderHints[right].Recursive {
			return !output.DecoderHints[left].Recursive
		}
		return slices.Compare(
			output.DecoderHints[left].DecoderIDs,
			output.DecoderHints[right].DecoderIDs,
		) < 0
	})
	return output
}

func (p SourcePlan) RequestedDecoderIDs(locator artifactstore.Locator) []artifactstore.DecoderID {
	seen := make(map[artifactstore.DecoderID]struct{})

	for _, hint := range p.DecoderHints {
		if !decoderHintMatchesLocator(hint, locator) {
			continue
		}
		for _, decoderID := range hint.DecoderIDs {
			seen[decoderID] = struct{}{}
		}
	}

	output := make([]artifactstore.DecoderID, 0, len(seen))
	for decoderID := range seen {
		output = append(output, decoderID)
	}
	slices.Sort(output)
	return output
}

func decoderHintMatchesLocator(
	hint DecoderHint,
	locator artifactstore.Locator,
) bool {
	if locator == hint.Locator {
		return true
	}
	if !hint.Recursive {
		if hint.Locator == "." {
			return !strings.Contains(string(locator), "/")
		}
		prefix := string(hint.Locator) + "/"
		relative, found := strings.CutPrefix(string(locator), prefix)
		return found && relative != "" && !strings.Contains(relative, "/")
	}
	if hint.Locator == "." {
		return locator != "."
	}
	return strings.HasPrefix(string(locator), string(hint.Locator)+"/")
}
