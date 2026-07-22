package contextadapter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
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

	defaultMaxContextPromptBytes   = 128 << 10
	defaultMaxContextDocumentBytes = 64 << 10
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
	if p.MaxPromptBytes <= 0 ||
		p.MaxPromptBytes > artifactstore.MaxDefinitionBodyBytes {
		return fmt.Errorf(
			"%w: Context prompt byte budget is invalid",
			engine.ErrInvalidWorkspace,
		)
	}
	if p.MaxDocumentBytes <= 0 ||
		p.MaxDocumentBytes > p.MaxPromptBytes {
		return fmt.Errorf(
			"%w: Context per-document byte budget is invalid",
			engine.ErrInvalidWorkspace,
		)
	}
	switch p.Overflow {
	case OverflowTruncate, OverflowExclude:
		return nil
	default:
		return fmt.Errorf(
			"%w: unsupported Context overflow behavior %q",
			engine.ErrInvalidWorkspace,
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

type CompositionDecision struct {
	RecordID      artifactstore.RecordID `json:"recordID"`
	Status        CompositionStatus      `json:"status"`
	Code          string                 `json:"code,omitempty"`
	OriginalBytes int                    `json:"originalBytes"`
	IncludedBytes int                    `json:"includedBytes"`
}

func applyCompositionPolicy(
	policy CompositionPolicy,
	values []ContextContribution,
	diagnostics []artifactstore.Diagnostic,
	decisions []CompositionDecision,
) (
	[]ContextContribution,
	string,
	[]artifactstore.Diagnostic,
	[]CompositionDecision,
) {
	policy = policy.Normalized()
	included := make([]ContextContribution, 0, len(values))
	var prompt strings.Builder

	for _, input := range values {
		value := input
		content := strings.TrimSpace(value.Content)
		originalBytes := len(content)
		status := CompositionIncluded
		code := ""

		if len(content) > policy.MaxDocumentBytes {
			if policy.Overflow == OverflowExclude {
				code = DiagnosticCodeContextDocumentExcluded
				diagnostics = artifactstore.AppendDiagnostics(
					diagnostics,
					compositionDiagnostic(
						value,
						code,
						fmt.Sprintf(
							"Context contribution exceeds the %d byte per-document limit",
							policy.MaxDocumentBytes,
						),
					),
				)
				decisions = append(decisions, CompositionDecision{
					RecordID:      value.RecordID,
					Status:        CompositionExcluded,
					Code:          code,
					OriginalBytes: originalBytes,
				})
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
				diagnostics = artifactstore.AppendDiagnostics(
					diagnostics,
					compositionDiagnostic(
						value,
						code,
						"Context contribution was excluded because the aggregate prompt budget was exhausted",
					),
				)
				decisions = append(decisions, CompositionDecision{
					RecordID:      value.RecordID,
					Status:        CompositionExcluded,
					Code:          code,
					OriginalBytes: originalBytes,
				})
				continue
			}

			emptyRendered := renderContextContribution(value, "")
			contentBudget := remaining - len(emptyRendered)
			if contentBudget <= 0 {
				code = DiagnosticCodeContextBudgetExceeded
				diagnostics = artifactstore.AppendDiagnostics(
					diagnostics,
					compositionDiagnostic(
						value,
						code,
						"Context contribution was excluded because no aggregate prompt capacity remained",
					),
				)
				decisions = append(decisions, CompositionDecision{
					RecordID:      value.RecordID,
					Status:        CompositionExcluded,
					Code:          code,
					OriginalBytes: originalBytes,
				})
				continue
			}
			content = truncateUTF8(content, contentBudget)
			if strings.TrimSpace(content) == "" {
				code = DiagnosticCodeContextBudgetExceeded
				diagnostics = artifactstore.AppendDiagnostics(
					diagnostics,
					compositionDiagnostic(
						value,
						code,
						"Context contribution was excluded because truncation left no usable content",
					),
				)
				decisions = append(decisions, CompositionDecision{
					RecordID:      value.RecordID,
					Status:        CompositionExcluded,
					Code:          code,
					OriginalBytes: originalBytes,
				})
				continue
			}
			rendered = renderContextContribution(value, content)
			status = CompositionTruncated
			code = DiagnosticCodeContextBudgetExceeded
		}

		if status == CompositionTruncated {
			diagnostics = artifactstore.AppendDiagnostics(
				diagnostics,
				compositionDiagnostic(
					value,
					code,
					"Context contribution was truncated to satisfy prompt composition limits",
				),
			)
		}

		if prompt.Len() > 0 {
			prompt.WriteString(contextPromptSeparator)
		}
		prompt.WriteString(rendered)
		value.OriginalBytes = originalBytes
		value.IncludedBytes = len(content)
		value.Truncated = status == CompositionTruncated
		value.Content = content
		included = append(included, value)
		decisions = append(decisions, CompositionDecision{
			RecordID:      value.RecordID,
			Status:        status,
			Code:          code,
			OriginalBytes: originalBytes,
			IncludedBytes: len(content),
		})
	}
	return included, prompt.String(), diagnostics, decisions
}

func renderContextContribution(value ContextContribution, content string) string {
	var output strings.Builder
	fmt.Fprintf(
		&output,
		contextPromptStartFormat,
		value.Name,
		value.Role,
		value.Locator,
	)
	output.WriteString(content)
	output.WriteString(contextPromptEndMarker)
	return output.String()
}

func compositionDiagnostic(
	value ContextContribution,
	code string,
	message string,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticWarning,
		Code:     code,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator: value.Locator,
		},
	}
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
