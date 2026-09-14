package runtime

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	defaultMaxContextPromptBytes   = 128 << 10
	defaultMaxContextDocumentBytes = 64 << 10
	maxContextPromptBytes          = 2 << 20

	contextPromptSeparator   = "\n\n"
	contextPromptStartFormat = "<<<WORKSPACE_ARTIFACT type=%q name=%q source=%q>>>\n"
	contextPromptEndMarker   = "\n<<<END_WORKSPACE_CONTEXT>>>"
)

type OverflowBehavior string

const (
	OverflowTruncate OverflowBehavior = "truncate"
	OverflowExclude  OverflowBehavior = "exclude"
)

const (
	DiagnosticCodeContextDocumentTruncated = "workspace.context.document-truncated"
	DiagnosticCodeContextDocumentExcluded  = "workspace.context.document-excluded"
	DiagnosticCodeContextBudgetExceeded    = "workspace.context.budget-exceeded"
)

type CompositionPolicy struct {
	MaxPromptBytes   int              `json:"maxPromptBytes"`
	MaxDocumentBytes int              `json:"maxDocumentBytes"`
	Overflow         OverflowBehavior `json:"overflow"`
}

func DefaultCompositionPolicy() CompositionPolicy {
	return CompositionPolicy{
		MaxPromptBytes:   defaultMaxContextPromptBytes,
		MaxDocumentBytes: defaultMaxContextDocumentBytes,
		Overflow:         OverflowTruncate,
	}
}

func (p CompositionPolicy) Normalized() CompositionPolicy {
	if p.MaxPromptBytes == 0 {
		p.MaxPromptBytes = defaultMaxContextPromptBytes
	}
	if p.MaxDocumentBytes == 0 {
		p.MaxDocumentBytes = defaultMaxContextDocumentBytes
	}
	if p.Overflow == "" {
		p.Overflow = OverflowTruncate
	}
	return p
}

func (p CompositionPolicy) Validate() error {
	p = p.Normalized()

	if p.MaxPromptBytes <= 0 || p.MaxPromptBytes > maxContextPromptBytes {
		return errors.New("workspace context prompt byte budget is invalid")
	}
	if p.MaxDocumentBytes <= 0 || p.MaxDocumentBytes > p.MaxPromptBytes {
		return errors.New("workspace context document byte budget is invalid")
	}

	switch p.Overflow {
	case OverflowTruncate, OverflowExclude:
		return nil
	default:
		return fmt.Errorf(
			"unsupported workspace context overflow behavior %q",
			p.Overflow,
		)
	}
}

type CompositionStatus string

const (
	CompositionIncluded    CompositionStatus = "included"
	CompositionTruncated   CompositionStatus = "truncated"
	CompositionExcluded    CompositionStatus = "excluded"
	CompositionDenied      CompositionStatus = "denied"
	CompositionUnavailable CompositionStatus = "unavailable"
)

// ContextContribution is an opaque runtime input. Artifact Store identity,
// source configuration, and persistence types deliberately do not enter this
// package.
type ContextContribution struct {
	ID      string
	Name    string
	Kind    string
	Locator string
	Content string

	OriginalBytes int
	IncludedBytes int
	Truncated     bool
}

type CompositionDiagnostic struct {
	ID      string
	Code    string
	Message string
}

type CompositionDecision struct {
	ID            string
	Status        CompositionStatus
	Code          string
	OriginalBytes int
	IncludedBytes int
}

type CompositionResult struct {
	Contributions []ContextContribution
	Prompt        string
	Diagnostics   []CompositionDiagnostic
	Decisions     []CompositionDecision
}

// Engine owns deterministic prompt budgeting, truncation, and rendering. It
// intentionally has no Artifact Store dependency.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Compose(
	policy CompositionPolicy,
	values []ContextContribution,
) (CompositionResult, error) {
	if e == nil {
		return CompositionResult{}, errors.New("workspace context runtime engine is nil")
	}
	if err := policy.Validate(); err != nil {
		return CompositionResult{}, err
	}
	policy = policy.Normalized()

	result := CompositionResult{
		Contributions: make([]ContextContribution, 0, len(values)),
		Diagnostics:   make([]CompositionDiagnostic, 0),
		Decisions:     make([]CompositionDecision, 0, len(values)),
	}

	seen := make(map[string]struct{}, len(values))
	var prompt strings.Builder

	for _, input := range values {
		if strings.TrimSpace(input.ID) == "" {
			return CompositionResult{}, errors.New(
				"workspace context contribution identity is required",
			)
		}
		if _, duplicate := seen[input.ID]; duplicate {
			return CompositionResult{}, fmt.Errorf(
				"duplicate workspace context contribution identity %q",
				input.ID,
			)
		}
		seen[input.ID] = struct{}{}

		value := input
		content := value.Content
		originalBytes := len(content)
		status := CompositionIncluded
		code := ""

		if len(content) > policy.MaxDocumentBytes {
			if policy.Overflow == OverflowExclude {
				code = DiagnosticCodeContextDocumentExcluded
				result.Diagnostics = append(
					result.Diagnostics,
					CompositionDiagnostic{
						ID:   value.ID,
						Code: code,
						Message: fmt.Sprintf(
							"Context contribution exceeds the %d byte per-document limit",
							policy.MaxDocumentBytes,
						),
					},
				)
				result.Decisions = append(
					result.Decisions,
					CompositionDecision{
						ID:            value.ID,
						Status:        CompositionExcluded,
						Code:          code,
						OriginalBytes: originalBytes,
					},
				)
				continue
			}

			content = truncateUTF8(content, policy.MaxDocumentBytes)
			status = CompositionTruncated
			code = DiagnosticCodeContextDocumentTruncated
		}

		separatorBytes := 0
		if prompt.Len() > 0 {
			separatorBytes = len(contextPromptSeparator)
		}

		rendered := renderContextContribution(value, content)
		remaining := policy.MaxPromptBytes - prompt.Len() - separatorBytes

		if len(rendered) > remaining {
			if policy.Overflow == OverflowExclude {
				code = DiagnosticCodeContextBudgetExceeded
				result.Diagnostics = append(
					result.Diagnostics,
					CompositionDiagnostic{
						ID:      value.ID,
						Code:    code,
						Message: "Context contribution was excluded because the aggregate prompt budget was exhausted",
					},
				)
				result.Decisions = append(
					result.Decisions,
					CompositionDecision{
						ID:            value.ID,
						Status:        CompositionExcluded,
						Code:          code,
						OriginalBytes: originalBytes,
					},
				)
				continue
			}

			emptyRendered := renderContextContribution(value, "")
			contentBudget := remaining - len(emptyRendered)
			if contentBudget <= 0 {
				code = DiagnosticCodeContextBudgetExceeded
				result.Diagnostics = append(
					result.Diagnostics,
					CompositionDiagnostic{
						ID:      value.ID,
						Code:    code,
						Message: "Context contribution was excluded because no aggregate prompt capacity remained",
					},
				)
				result.Decisions = append(
					result.Decisions,
					CompositionDecision{
						ID:            value.ID,
						Status:        CompositionExcluded,
						Code:          code,
						OriginalBytes: originalBytes,
					},
				)
				continue
			}

			content = truncateUTF8(content, contentBudget)
			if strings.TrimSpace(content) == "" {
				code = DiagnosticCodeContextBudgetExceeded
				result.Diagnostics = append(
					result.Diagnostics,
					CompositionDiagnostic{
						ID:      value.ID,
						Code:    code,
						Message: "Context contribution was excluded because truncation left no usable content",
					},
				)
				result.Decisions = append(
					result.Decisions,
					CompositionDecision{
						ID:            value.ID,
						Status:        CompositionExcluded,
						Code:          code,
						OriginalBytes: originalBytes,
					},
				)
				continue
			}

			rendered = renderContextContribution(value, content)
			status = CompositionTruncated
			code = DiagnosticCodeContextBudgetExceeded
		}

		if status == CompositionTruncated {
			result.Diagnostics = append(
				result.Diagnostics,
				CompositionDiagnostic{
					ID:      value.ID,
					Code:    code,
					Message: "Context contribution was truncated to satisfy prompt composition limits",
				},
			)
		}

		if prompt.Len() > 0 {
			prompt.WriteString(contextPromptSeparator)
		}
		prompt.WriteString(rendered)

		value.Content = content
		value.OriginalBytes = originalBytes
		value.IncludedBytes = len(content)
		value.Truncated = status == CompositionTruncated

		result.Contributions = append(result.Contributions, value)
		result.Decisions = append(
			result.Decisions,
			CompositionDecision{
				ID:            value.ID,
				Status:        status,
				Code:          code,
				OriginalBytes: originalBytes,
				IncludedBytes: len(content),
			},
		)
	}

	result.Prompt = prompt.String()
	return result, nil
}

func renderContextContribution(
	value ContextContribution,
	content string,
) string {
	var output strings.Builder

	fmt.Fprintf(
		&output,
		contextPromptStartFormat,
		value.Kind,
		value.Name,
		value.Locator,
	)
	output.WriteString(content)
	output.WriteString(contextPromptEndMarker)

	return output.String()
}

func truncateUTF8(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if len(value) <= maximum {
		return value
	}

	value = value[:maximum]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
