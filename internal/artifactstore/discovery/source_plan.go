package discovery

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// DirectoryRoot is generic source-discovery scope shared by planners,
// preferences, and executable SourcePlans.
type DirectoryRoot struct {
	Root            basespec.Locator `json:"root"`
	Recursive       bool             `json:"recursive"`
	IncludePatterns []string         `json:"includePatterns,omitempty"`
}

func (r DirectoryRoot) Validate() error {
	if err := basespec.ValidateLocator(r.Root, true); err != nil {
		return err
	}
	seenPatterns := make(map[string]struct{}, len(r.IncludePatterns))
	for _, pattern := range r.IncludePatterns {
		if err := ValidateIncludePattern(pattern); err != nil {
			return err
		}
		if _, duplicate := seenPatterns[pattern]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery pattern %q",
				basespec.ErrInvalid,
				pattern,
			)
		}
		seenPatterns[pattern] = struct{}{}
	}
	return nil
}

// ValidateIncludePattern validates a source-relative glob. It deliberately
// rejects path traversal and host-path syntax before passing the pattern to
// path.Match.
func ValidateIncludePattern(pattern string) error {
	if err := basespec.ValidateRequiredText(
		"discovery pattern",
		pattern,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if strings.HasPrefix(pattern, "/") ||
		strings.ContainsAny(pattern, `\:`) {
		return fmt.Errorf(
			"%w: discovery pattern contains a disallowed path character",
			basespec.ErrInvalid,
		)
	}
	for segment := range strings.SplitSeq(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf(
				"%w: discovery pattern contains an invalid path segment",
				basespec.ErrInvalid,
			)
		}
	}
	if _, err := path.Match(pattern, "candidate"); err != nil {
		return fmt.Errorf(
			"%w: invalid discovery pattern %q: %w",
			basespec.ErrInvalid,
			pattern,
			err,
		)
	}
	return nil
}

type DecoderHint struct {
	Locator    basespec.Locator     `json:"locator"`
	Recursive  bool                 `json:"recursive"`
	DecoderIDs []basespec.DecoderID `json:"decoderIDs"`
}

type decoderHintScope struct {
	Locator   basespec.Locator
	Recursive bool
}

type SourcePlan struct {
	SourceID               basespec.SourceID                      `json:"sourceID"`
	ExplicitLocators       []basespec.Locator                     `json:"explicitLocators,omitempty"`
	DirectoryRoots         []DirectoryRoot                        `json:"directoryRoots,omitempty"`
	DecoderHints           []DecoderHint                          `json:"decoderHints,omitempty"`
	ExpectedContentDigests map[basespec.Locator]cryptoutil.Digest `json:"expectedContentDigests,omitempty"`
	ExpectedGeneration     string                                 `json:"expectedGeneration,omitempty"`
	AllowedDecoderIDs      []basespec.DecoderID                   `json:"allowedDecoderIDs,omitempty"`
	Authoritative          bool                                   `json:"authoritative"`
	MaxCandidateBytes      int64                                  `json:"maxCandidateBytes"`
	MaxTotalBytes          int64                                  `json:"maxTotalBytes"`
	MaxCandidates          int                                    `json:"maxCandidates"`
	MaxEntries             int                                    `json:"maxEntries"`
	MaxDepth               int                                    `json:"maxDepth"`
}

func (p SourcePlan) Validate() error {
	if err := basespec.ValidateSourceID(p.SourceID); err != nil {
		return err
	}
	if len(p.ExplicitLocators) == 0 &&
		len(p.DirectoryRoots) == 0 {
		return fmt.Errorf(
			"%w: source discovery plan has no scope",
			basespec.ErrInvalid,
		)
	}
	if p.ExpectedGeneration != "" {
		if err := basespec.ValidateSourceGeneration(
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
			basespec.ErrInvalid,
		)
	}
	if p.MaxCandidateBytes > basespec.MaxCandidateBytes ||
		p.MaxTotalBytes > basespec.MaxScanBytes ||
		p.MaxCandidates > basespec.MaxDiscoveryCandidates ||
		p.MaxEntries > basespec.MaxDiscoveryEntries ||
		p.MaxDepth > basespec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: discovery limits exceed hard safety limits",
			basespec.ErrInvalid,
		)
	}
	seenLocators := make(map[basespec.Locator]struct{}, len(p.ExplicitLocators))
	for _, locator := range p.ExplicitLocators {
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate explicit discovery locator %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenRoots := make(map[basespec.Locator]struct{}, len(p.DirectoryRoots))
	for _, root := range p.DirectoryRoots {
		if err := root.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery root %q",
				basespec.ErrInvalid,
				root.Root,
			)
		}
		seenRoots[root.Root] = struct{}{}
	}

	if len(p.ExpectedContentDigests) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: expected content digests exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDiscoveryCandidates,
		)
	}
	for locator, digest := range p.ExpectedContentDigests {
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(digest); err != nil {
			return err
		}
		if !locatorInScope(locator, p) {
			return fmt.Errorf(
				"%w: expected content digest locator %q is outside discovery scope",
				basespec.ErrInvalid,
				locator,
			)
		}
	}

	seenHints := make(map[decoderHintScope]struct{}, len(p.DecoderHints))
	for index, hint := range p.DecoderHints {
		scope := decoderHintScope{Locator: hint.Locator, Recursive: hint.Recursive}
		if err := basespec.ValidateLocator(hint.Locator, true); err != nil {
			return fmt.Errorf("decoder hint %d: %w", index, err)
		}
		if len(hint.DecoderIDs) == 0 {
			return fmt.Errorf(
				"%w: decoder hint %d has no decoder IDs",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := seenHints[scope]; duplicate {
			return fmt.Errorf(
				"%w: duplicate decoder hint scope %q recursive=%t",
				basespec.ErrInvalid,
				hint.Locator,
				hint.Recursive,
			)
		}
		seenHints[scope] = struct{}{}

		seenHintDecoders := make(map[basespec.DecoderID]struct{}, len(hint.DecoderIDs))
		for _, decoderID := range hint.DecoderIDs {
			if err := basespec.ValidateDecoderID(decoderID); err != nil {
				return err
			}
			if _, duplicate := seenHintDecoders[decoderID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate decoder hint ID %q",
					basespec.ErrInvalid,
					decoderID,
				)
			}
			seenHintDecoders[decoderID] = struct{}{}
		}
	}

	seenDecoders := make(map[basespec.DecoderID]struct{}, len(p.AllowedDecoderIDs))
	for _, decoderID := range p.AllowedDecoderIDs {
		if err := basespec.ValidateDecoderID(decoderID); err != nil {
			return err
		}
		if _, duplicate := seenDecoders[decoderID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate allowed decoder %q",
				basespec.ErrInvalid,
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
		[]basespec.Locator(nil),
		p.ExplicitLocators...,
	)
	output.ExpectedContentDigests = maps.Clone(p.ExpectedContentDigests)
	output.AllowedDecoderIDs = append(
		[]basespec.DecoderID(nil),
		p.AllowedDecoderIDs...,
	)
	output.DecoderHints = make([]DecoderHint, len(p.DecoderHints))
	for index, hint := range p.DecoderHints {
		output.DecoderHints[index] = hint
		output.DecoderHints[index].DecoderIDs = append(
			[]basespec.DecoderID(nil),
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
		sort.Strings(output.DirectoryRoots[index].IncludePatterns)
	}

	if output.MaxCandidateBytes == 0 {
		output.MaxCandidateBytes = basespec.MaxCandidateBytes
	}
	if output.MaxTotalBytes == 0 {
		output.MaxTotalBytes = basespec.MaxScanBytes
	}
	if output.MaxCandidates == 0 {
		output.MaxCandidates = basespec.DefaultMaxCandidates
	}
	if output.MaxEntries == 0 {
		output.MaxEntries = basespec.DefaultMaxEntries
	}
	if output.MaxDepth == 0 {
		output.MaxDepth = basespec.DefaultMaxDepth
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

func (p SourcePlan) RequestedDecoderIDs(locator basespec.Locator) []basespec.DecoderID {
	seen := make(map[basespec.DecoderID]struct{})

	for _, hint := range p.DecoderHints {
		if !decoderHintMatchesLocator(hint, locator) {
			continue
		}
		for _, decoderID := range hint.DecoderIDs {
			seen[decoderID] = struct{}{}
		}
	}

	output := make([]basespec.DecoderID, 0, len(seen))
	for decoderID := range seen {
		output = append(output, decoderID)
	}
	slices.Sort(output)
	return output
}

func decoderHintMatchesLocator(
	hint DecoderHint,
	locator basespec.Locator,
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
