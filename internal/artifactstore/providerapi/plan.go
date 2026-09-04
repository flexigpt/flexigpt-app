package providerapi

import (
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Plan is a provider-owned declaration of source discovery scope.
//
// Artifact Store validates and converts this contract into its internal
// discovery.Plan before it opens source snapshots or publishes a catalog.
type Plan struct {
	// Revision identifies provider-owned discovery behavior. Artifact Store
	// includes it in the catalog plan fingerprint.
	Revision string       `json:"revision,omitempty"`
	Sources  []SourcePlan `json:"sources"`
}

func (p Plan) Clone() Plan {
	output := p
	output.Sources = make([]SourcePlan, len(p.Sources))
	for index, value := range p.Sources {
		output.Sources[index] = value.Clone()
	}
	return output
}

// Normalized returns an independently owned deterministic plan.
func (p Plan) Normalized() Plan {
	output := p.Clone()
	for index := range output.Sources {
		output.Sources[index] = output.Sources[index].Normalized()
	}
	sort.Slice(output.Sources, func(left, right int) bool {
		return output.Sources[left].SourceID <
			output.Sources[right].SourceID
	})
	return output
}

func (p Plan) Validate() error {
	if err := basespec.ValidateOptionalText(
		"provider discovery plan revision",
		p.Revision,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}

	seen := make(map[basespec.SourceID]struct{}, len(p.Sources))
	for index, sourcePlan := range p.Sources {
		if err := sourcePlan.Validate(); err != nil {
			return fmt.Errorf(
				"provider source plan %d: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[sourcePlan.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider source plan for %q",
				basespec.ErrInvalid,
				sourcePlan.SourceID,
			)
		}
		seen[sourcePlan.SourceID] = struct{}{}
	}

	return nil
}

// DirectoryRoot declares one source-relative directory discovery scope.
type DirectoryRoot struct {
	Root            basespec.Locator `json:"root"`
	Recursive       bool             `json:"recursive"`
	IncludePatterns []string         `json:"includePatterns,omitempty"`
}

func (r DirectoryRoot) Clone() DirectoryRoot {
	output := r
	output.IncludePatterns = append(
		[]string(nil),
		r.IncludePatterns...,
	)
	return output
}

func (r DirectoryRoot) Validate() error {
	if err := basespec.ValidateLocator(r.Root, true); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(r.IncludePatterns))
	for _, pattern := range r.IncludePatterns {
		if err := basespec.ValidateIncludePattern(pattern); err != nil {
			return err
		}
		if _, duplicate := seen[pattern]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider discovery pattern %q",
				basespec.ErrInvalid,
				pattern,
			)
		}
		seen[pattern] = struct{}{}
	}

	return nil
}

// DecoderHint identifies decoders preferred for a source-relative scope.
type DecoderHint struct {
	Locator    basespec.Locator     `json:"locator"`
	Recursive  bool                 `json:"recursive"`
	DecoderIDs []basespec.DecoderID `json:"decoderIDs"`
}

func (h DecoderHint) Clone() DecoderHint {
	output := h
	output.DecoderIDs = append(
		[]basespec.DecoderID(nil),
		h.DecoderIDs...,
	)
	return output
}

type decoderHintScope struct {
	Locator   basespec.Locator
	Recursive bool
}

// SourcePlan declares discovery scope for one attached Source.
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

func (p SourcePlan) Clone() SourcePlan {
	output := p
	output.ExplicitLocators = append(
		[]basespec.Locator(nil),
		p.ExplicitLocators...,
	)
	output.DirectoryRoots = make(
		[]DirectoryRoot,
		len(p.DirectoryRoots),
	)
	for index, root := range p.DirectoryRoots {
		output.DirectoryRoots[index] = root.Clone()
	}
	output.DecoderHints = make(
		[]DecoderHint,
		len(p.DecoderHints),
	)
	for index, hint := range p.DecoderHints {
		output.DecoderHints[index] = hint.Clone()
	}
	output.ExpectedContentDigests = maps.Clone(
		p.ExpectedContentDigests,
	)
	output.AllowedDecoderIDs = append(
		[]basespec.DecoderID(nil),
		p.AllowedDecoderIDs...,
	)
	return output
}

func (p SourcePlan) Validate() error {
	if err := basespec.ValidateSourceID(p.SourceID); err != nil {
		return err
	}
	if len(p.ExplicitLocators) == 0 &&
		len(p.DirectoryRoots) == 0 {
		return fmt.Errorf(
			"%w: provider source discovery plan has no scope",
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
			"%w: provider discovery limits cannot be negative",
			basespec.ErrInvalid,
		)
	}
	if p.MaxCandidateBytes > basespec.MaxCandidateBytes ||
		p.MaxTotalBytes > basespec.MaxScanBytes ||
		p.MaxCandidates > basespec.MaxDiscoveryCandidates ||
		p.MaxEntries > basespec.MaxDiscoveryEntries ||
		p.MaxDepth > basespec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: provider discovery limits exceed hard safety limits",
			basespec.ErrInvalid,
		)
	}

	seenLocators := make(
		map[basespec.Locator]struct{},
		len(p.ExplicitLocators),
	)
	for _, locator := range p.ExplicitLocators {
		if err := basespec.ValidateLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider explicit locator %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenRoots := make(
		map[basespec.Locator]struct{},
		len(p.DirectoryRoots),
	)
	for _, root := range p.DirectoryRoots {
		if err := root.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider discovery root %q",
				basespec.ErrInvalid,
				root.Root,
			)
		}
		seenRoots[root.Root] = struct{}{}
	}

	if len(p.ExpectedContentDigests) >
		basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: provider expected content digest count exceeds limit",
			basespec.ErrInvalid,
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
				"%w: provider expected digest locator %q is outside discovery scope",
				basespec.ErrInvalid,
				locator,
			)
		}
	}

	seenHints := make(
		map[decoderHintScope]struct{},
		len(p.DecoderHints),
	)
	for index, hint := range p.DecoderHints {
		scope := decoderHintScope{
			Locator:   hint.Locator,
			Recursive: hint.Recursive,
		}
		if err := basespec.ValidateLocator(hint.Locator, true); err != nil {
			return fmt.Errorf(
				"provider decoder hint %d: %w",
				index,
				err,
			)
		}
		if len(hint.DecoderIDs) == 0 {
			return fmt.Errorf(
				"%w: provider decoder hint %d has no decoder IDs",
				basespec.ErrInvalid,
				index,
			)
		}
		if _, duplicate := seenHints[scope]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider decoder hint scope %q recursive=%t",
				basespec.ErrInvalid,
				hint.Locator,
				hint.Recursive,
			)
		}
		seenHints[scope] = struct{}{}

		seenDecoderIDs := make(
			map[basespec.DecoderID]struct{},
			len(hint.DecoderIDs),
		)
		for _, decoderID := range hint.DecoderIDs {
			if err := basespec.ValidateDecoderID(decoderID); err != nil {
				return err
			}
			if _, duplicate := seenDecoderIDs[decoderID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate provider decoder hint ID %q",
					basespec.ErrInvalid,
					decoderID,
				)
			}
			seenDecoderIDs[decoderID] = struct{}{}
		}
	}

	seenAllowed := make(
		map[basespec.DecoderID]struct{},
		len(p.AllowedDecoderIDs),
	)
	for _, decoderID := range p.AllowedDecoderIDs {
		if err := basespec.ValidateDecoderID(decoderID); err != nil {
			return err
		}
		if _, duplicate := seenAllowed[decoderID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate provider allowed decoder %q",
				basespec.ErrInvalid,
				decoderID,
			)
		}
		seenAllowed[decoderID] = struct{}{}
	}

	return nil
}

// Normalized returns an independently owned deterministic SourcePlan with
// Artifact Store discovery defaults applied.
func (p SourcePlan) Normalized() SourcePlan {
	output := p.Clone()

	for index := range output.DirectoryRoots {
		sort.Strings(output.DirectoryRoots[index].IncludePatterns)
	}
	for index := range output.DecoderHints {
		slices.Sort(output.DecoderHints[index].DecoderIDs)
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
	slices.Sort(output.AllowedDecoderIDs)
	sort.Slice(output.DirectoryRoots, func(left, right int) bool {
		return output.DirectoryRoots[left].Root <
			output.DirectoryRoots[right].Root
	})
	sort.Slice(output.DecoderHints, func(left, right int) bool {
		leftValue := output.DecoderHints[left]
		rightValue := output.DecoderHints[right]
		if leftValue.Locator != rightValue.Locator {
			return leftValue.Locator < rightValue.Locator
		}
		if leftValue.Recursive != rightValue.Recursive {
			return !leftValue.Recursive
		}
		return slices.Compare(
			leftValue.DecoderIDs,
			rightValue.DecoderIDs,
		) < 0
	})

	return output
}

func (p SourcePlan) RequestedDecoderIDs(
	locator basespec.Locator,
) []basespec.DecoderID {
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

func locatorInScope(
	locator basespec.Locator,
	plan SourcePlan,
) bool {
	if slices.Contains(plan.ExplicitLocators, locator) {
		return true
	}
	for _, root := range plan.DirectoryRoots {
		if matchesDirectoryRoot(root, locator) {
			return true
		}
	}
	return false
}

func matchesDirectoryRoot(
	root DirectoryRoot,
	locator basespec.Locator,
) bool {
	base := string(root.Root)
	value := string(locator)
	relative := value

	if base != "." {
		prefix := base + "/"
		if !strings.HasPrefix(value, prefix) {
			return false
		}
		relative = strings.TrimPrefix(value, prefix)
	}

	if !root.Recursive && strings.Contains(relative, "/") {
		return false
	}
	if len(root.IncludePatterns) == 0 {
		return true
	}

	for _, pattern := range root.IncludePatterns {
		if matched, _ := path.Match(pattern, relative); matched {
			return true
		}
		if !strings.Contains(pattern, "/") {
			if matched, _ := path.Match(
				pattern,
				path.Base(relative),
			); matched {
				return true
			}
		}
	}

	return false
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
		relative, found := strings.CutPrefix(
			string(locator),
			prefix,
		)
		return found &&
			relative != "" &&
			!strings.Contains(relative, "/")
	}
	if hint.Locator == "." {
		return locator != "."
	}
	return strings.HasPrefix(
		string(locator),
		string(hint.Locator)+"/",
	)
}

// Fingerprint returns the same semantic discovery fingerprint shape used by
// Artifact Store's internal discovery.Plan. ExpectedGeneration is deliberately
// excluded because it is a concurrency token rather than a capability input.
func (p Plan) Fingerprint() (cryptoutil.Digest, error) {
	p = p.Normalized()
	if err := p.Validate(); err != nil {
		return "", err
	}

	values := make([]SourcePlan, len(p.Sources))
	for index, value := range p.Sources {
		values[index] = value.Normalized()
		values[index].ExpectedGeneration = ""
	}

	raw, err := json.Marshal(struct {
		Revision string       `json:"revision,omitempty"`
		Sources  []SourcePlan `json:"sources"`
	}{
		Revision: p.Revision,
		Sources:  values,
	})
	if err != nil {
		return "", err
	}

	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}
