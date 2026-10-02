package source

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// DirectoryRoot declares one source-relative declaration discovery scope.
//
// IncludePatterns and ExcludePatterns are evaluated relative to Root.
// Exclusions are evaluated after inclusions.
type DirectoryRoot struct {
	Root            model.Locator `json:"root"`
	Recursive       bool          `json:"recursive,omitempty"`
	IncludePatterns []string      `json:"includePatterns,omitempty"`
	ExcludePatterns []string      `json:"excludePatterns,omitempty"`
}

func (r DirectoryRoot) Clone() DirectoryRoot {
	output := r
	output.IncludePatterns = append(
		[]string(nil),
		r.IncludePatterns...,
	)
	output.ExcludePatterns = append(
		[]string(nil),
		r.ExcludePatterns...,
	)
	return output
}

func (r DirectoryRoot) Validate() error {
	if err := r.Root.Validate(true); err != nil {
		return err
	}
	if err := model.ValidatePathPatterns(
		"Source discovery include patterns",
		r.IncludePatterns,
	); err != nil {
		return err
	}
	return model.ValidatePathPatterns(
		"Source discovery exclude patterns",
		r.ExcludePatterns,
	)
}

// Matches reports whether locator is selected by this directory scope.
func (r DirectoryRoot) Matches(
	locator model.Locator,
) (bool, error) {
	if err := r.Root.Validate(true); err != nil {
		return false, err
	}
	if err := locator.Validate(false); err != nil {
		return false, err
	}
	selection, err := model.NewPathSelection(
		r.IncludePatterns,
		r.ExcludePatterns,
	)
	if err != nil {
		return false, err
	}

	value := string(locator)
	relative := value
	if r.Root != "." {
		prefix := string(r.Root) + "/"
		if !strings.HasPrefix(value, prefix) {
			return false, nil
		}
		relative = strings.TrimPrefix(value, prefix)
	}
	if relative == "" {
		return false, nil
	}
	if !r.Recursive && strings.Contains(relative, "/") {
		return false, nil
	}

	return selection.Match(relative)
}

// DecoderHint requests one or more decoders for a source-relative scope.
//
// Hints do not override AllowedDecoderIDs. The discovery engine checks both
// before invoking a decoder.
type DecoderHint struct {
	Locator    model.Locator     `json:"locator"`
	Recursive  bool              `json:"recursive,omitempty"`
	DecoderIDs []model.DecoderID `json:"decoderIDs"`
}

func (h DecoderHint) Clone() DecoderHint {
	output := h
	output.DecoderIDs = append(
		[]model.DecoderID(nil),
		h.DecoderIDs...,
	)
	return output
}

func (h DecoderHint) Validate() error {
	if err := h.Locator.Validate(true); err != nil {
		return err
	}
	if len(h.DecoderIDs) == 0 {
		return fmt.Errorf(
			"%w: Source decoder hint requires at least one decoder ID",
			model.ErrInvalid,
		)
	}

	for index, decoderID := range h.DecoderIDs {
		if err := decoderID.Validate(); err != nil {
			return fmt.Errorf(
				"Source decoder hint decoderIDs[%d]: %w",
				index,
				err,
			)
		}
	}
	return nil
}

// DiscoverySpec is Store-owned declaration discovery configuration for one
// Source. It remains separate from adapter-owned Source.Config.
//
// A zero DiscoverySpec is valid. Such a Source is available for Resource
// access but is skipped by Root refresh.
type DiscoverySpec struct {
	ExplicitLocators       []model.Locator                     `json:"explicitLocators,omitempty"`
	DirectoryRoots         []DirectoryRoot                     `json:"directoryRoots,omitempty"`
	DecoderHints           []DecoderHint                       `json:"decoderHints,omitempty"`
	AllowedDecoderIDs      []model.DecoderID                   `json:"allowedDecoderIDs,omitempty"`
	ExpectedContentDigests map[model.Locator]cryptoutil.Digest `json:"expectedContentDigests,omitempty"`
	Authoritative          bool                                `json:"authoritative,omitempty"`

	MaxCandidateBytes int64 `json:"maxCandidateBytes,omitempty"`
	MaxTotalBytes     int64 `json:"maxTotalBytes,omitempty"`
	MaxCandidates     int   `json:"maxCandidates,omitempty"`
	MaxEntries        int   `json:"maxEntries,omitempty"`
	MaxDepth          int   `json:"maxDepth,omitempty"`
}

func (s DiscoverySpec) Empty() bool {
	return len(s.ExplicitLocators) == 0 &&
		len(s.DirectoryRoots) == 0
}

func (s DiscoverySpec) Clone() DiscoverySpec {
	output := s
	output.ExplicitLocators = append(
		[]model.Locator(nil),
		s.ExplicitLocators...,
	)
	output.DirectoryRoots = make(
		[]DirectoryRoot,
		len(s.DirectoryRoots),
	)
	for index, root := range s.DirectoryRoots {
		output.DirectoryRoots[index] = root.Clone()
	}
	output.DecoderHints = make(
		[]DecoderHint,
		len(s.DecoderHints),
	)
	for index, hint := range s.DecoderHints {
		output.DecoderHints[index] = hint.Clone()
	}
	output.AllowedDecoderIDs = append(
		[]model.DecoderID(nil),
		s.AllowedDecoderIDs...,
	)
	output.ExpectedContentDigests = maps.Clone(
		s.ExpectedContentDigests,
	)
	return output
}

// Normalized returns independently owned deterministic configuration.
//
// These arrays describe discovery configuration, not typed artifact
// composition. Sorting them therefore does not alter declaration semantics.
func (s DiscoverySpec) Normalized() DiscoverySpec {
	output := s.Clone()

	slices.Sort(output.ExplicitLocators)
	slices.Sort(output.AllowedDecoderIDs)

	for index := range output.DirectoryRoots {
		sort.Strings(output.DirectoryRoots[index].IncludePatterns)
		sort.Strings(output.DirectoryRoots[index].ExcludePatterns)
	}
	sort.Slice(output.DirectoryRoots, func(left, right int) bool {
		leftValue := output.DirectoryRoots[left]
		rightValue := output.DirectoryRoots[right]
		if leftValue.Root != rightValue.Root {
			return leftValue.Root < rightValue.Root
		}
		if leftValue.Recursive != rightValue.Recursive {
			return !leftValue.Recursive
		}
		includeOrder := slices.Compare(
			leftValue.IncludePatterns,
			rightValue.IncludePatterns,
		)
		if includeOrder != 0 {
			return includeOrder < 0
		}
		return slices.Compare(
			leftValue.ExcludePatterns,
			rightValue.ExcludePatterns,
		) < 0
	})

	for index := range output.DecoderHints {
		slices.Sort(output.DecoderHints[index].DecoderIDs)
	}
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

// Effective applies Store-owned bounded defaults to non-empty discovery.
func (s DiscoverySpec) Effective() DiscoverySpec {
	output := s.Normalized()
	if output.Empty() {
		return output
	}
	if output.MaxCandidateBytes == 0 {
		output.MaxCandidateBytes = model.MaxCandidateBytes
	}
	if output.MaxTotalBytes == 0 {
		output.MaxTotalBytes = model.MaxScanBytes
	}
	if output.MaxCandidates == 0 {
		output.MaxCandidates = model.DefaultMaxCandidates
	}
	if output.MaxEntries == 0 {
		output.MaxEntries = model.DefaultMaxEntries
	}
	if output.MaxDepth == 0 {
		output.MaxDepth = model.DefaultMaxDepth
	}
	return output
}

func (s DiscoverySpec) Validate() error {
	if s.MaxCandidateBytes < 0 ||
		s.MaxTotalBytes < 0 ||
		s.MaxCandidates < 0 ||
		s.MaxEntries < 0 ||
		s.MaxDepth < 0 {
		return fmt.Errorf(
			"%w: Source discovery limits cannot be negative",
			model.ErrInvalid,
		)
	}
	if s.MaxCandidateBytes > model.MaxCandidateBytes ||
		s.MaxTotalBytes > model.MaxScanBytes ||
		s.MaxCandidates > model.MaxDiscoveryCandidates ||
		s.MaxEntries > model.MaxDiscoveryEntries ||
		s.MaxDepth > model.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: Source discovery limits exceed Store safety limits",
			model.ErrInvalid,
		)
	}

	if s.Empty() {
		if len(s.DecoderHints) != 0 ||
			len(s.AllowedDecoderIDs) != 0 ||
			len(s.ExpectedContentDigests) != 0 ||
			s.Authoritative {
			return fmt.Errorf(
				"%w: empty Source discovery cannot contain decoder, digest, or authority configuration",
				model.ErrInvalid,
			)
		}
		return nil
	}

	if len(s.ExplicitLocators) >
		model.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: Source explicit locator count exceeds limit",
			model.ErrInvalid,
		)
	}
	if len(s.DirectoryRoots) >
		model.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: Source directory root count exceeds limit",
			model.ErrInvalid,
		)
	}
	if len(s.DecoderHints) > model.MaxDecoderHints {
		return fmt.Errorf(
			"%w: Source decoder hint count exceeds limit",
			model.ErrInvalid,
		)
	}

	for index, locator := range s.ExplicitLocators {
		if err := locator.Validate(false); err != nil {
			return fmt.Errorf(
				"Source explicitLocators[%d]: %w",
				index,
				err,
			)
		}
	}

	for index, root := range s.DirectoryRoots {
		if err := root.Validate(); err != nil {
			return fmt.Errorf(
				"Source directoryRoots[%d]: %w",
				index,
				err,
			)
		}
	}

	for index, hint := range s.DecoderHints {
		if err := hint.Validate(); err != nil {
			return fmt.Errorf(
				"Source decoderHints[%d]: %w",
				index,
				err,
			)
		}
	}

	for index, decoderID := range s.AllowedDecoderIDs {
		if err := decoderID.Validate(); err != nil {
			return fmt.Errorf(
				"Source allowedDecoderIDs[%d]: %w",
				index,
				err,
			)
		}
	}

	if len(s.ExpectedContentDigests) >
		model.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: Source expected content digest count exceeds limit",
			model.ErrInvalid,
		)
	}
	for locator, digest := range s.ExpectedContentDigests {
		if err := locator.Validate(false); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(digest); err != nil {
			return fmt.Errorf(
				"Source expected content digest for %q: %w",
				locator,
				err,
			)
		}
		inScope, err := s.InScope(locator)
		if err != nil {
			return err
		}
		if !inScope {
			return fmt.Errorf(
				"%w: Source expected digest locator %q is outside discovery scope",
				model.ErrInvalid,
				locator,
			)
		}
	}

	return nil
}

// InScope reports whether a regular Source entry is selected by this
// discovery configuration.
func (s DiscoverySpec) InScope(
	locator model.Locator,
) (bool, error) {
	if err := locator.Validate(false); err != nil {
		return false, err
	}
	if slices.Contains(s.ExplicitLocators, locator) {
		return true, nil
	}
	for _, root := range s.DirectoryRoots {
		matched, err := root.Matches(locator)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// RequestedDecoderIDs returns a deterministic union of decoder hints
// matching locator.
func (s DiscoverySpec) RequestedDecoderIDs(
	locator model.Locator,
) []model.DecoderID {
	seen := make(map[model.DecoderID]struct{})
	for _, hint := range s.DecoderHints {
		if !decoderHintMatchesLocator(hint, locator) {
			continue
		}
		for _, decoderID := range hint.DecoderIDs {
			seen[decoderID] = struct{}{}
		}
	}

	output := make([]model.DecoderID, 0, len(seen))
	for decoderID := range seen {
		output = append(output, decoderID)
	}
	slices.Sort(output)
	return output
}

func (s DiscoverySpec) Fingerprint() (
	cryptoutil.Digest,
	error,
) {
	effective := s.Effective()
	if err := effective.Validate(); err != nil {
		return "", err
	}
	return cryptoutil.CanonicalDigest(effective)
}

func (s DiscoverySpec) Equal(
	other DiscoverySpec,
) bool {
	return reflect.DeepEqual(
		s.Effective(),
		other.Effective(),
	)
}

func decoderHintMatchesLocator(
	hint DecoderHint,
	locator model.Locator,
) bool {
	if locator == hint.Locator {
		return true
	}
	if hint.Locator == "." {
		if hint.Recursive {
			return locator != "."
		}
		return !strings.Contains(string(locator), "/")
	}

	prefix := string(hint.Locator) + "/"
	relative, found := strings.CutPrefix(
		string(locator),
		prefix,
	)
	if !found || relative == "" {
		return false
	}
	return hint.Recursive ||
		!strings.Contains(relative, "/")
}
