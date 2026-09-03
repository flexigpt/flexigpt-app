package runtime

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

const defaultApprovalTTL = 5 * time.Minute

type approvalDecisionKey struct {
	Server     mcpSpec.ServerID    `json:"server"`
	ToolName   string              `json:"toolName"`
	ToolDigest string              `json:"toolDigest,omitempty"`
	Risk       MCPToolRisk         `json:"risk"`
	Source     MCPInvocationSource `json:"source"`
	AppID      string              `json:"appInstanceID,omitempty"`
}

type pendingApproval struct {
	ID        string
	Token     string
	Summary   MCPApprovalSummary
	ExpiresAt time.Time
	Issued    bool
	Consumed  bool
}

type rememberedApprovalDecision struct {
	Summary    MCPApprovalSummary
	Resolution MCPApprovalResolution
}

type ApprovalManager struct {
	mu        sync.Mutex
	ttl       time.Duration
	pending   map[string]*pendingApproval
	decisions map[string]rememberedApprovalDecision
}

func NewApprovalManager(ttl time.Duration) *ApprovalManager {
	if ttl <= 0 {
		ttl = defaultApprovalTTL
	}
	return &ApprovalManager{
		ttl:       ttl,
		pending:   make(map[string]*pendingApproval),
		decisions: make(map[string]rememberedApprovalDecision),
	}
}

func (m *ApprovalManager) Create(
	ctx context.Context,
	summary MCPApprovalSummary,
) (string, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return "", err
	}
	if m == nil {
		return "", fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			ErrMCPRuntimeNotReady,
		)
	}
	if err := validateApprovalSummary(summary); err != nil {
		return "", err
	}

	id, err := randomApprovalToken(24)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pending == nil {
		m.pending = make(map[string]*pendingApproval)
	}
	m.purgeExpiredLocked(time.Now().UTC())
	m.pending[id] = &pendingApproval{
		ID:        id,
		Summary:   cloneApprovalSummary(summary),
		ExpiresAt: time.Now().UTC().Add(m.ttl),
	}
	return id, nil
}

func (m *ApprovalManager) Resolve(
	ctx context.Context,
	id string,
	resolution MCPApprovalResolution,
) (MCPApprovalResolutionResult, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return MCPApprovalResolutionResult{}, err
	}
	if m == nil {
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			ErrMCPRuntimeNotReady,
		)
	}
	if strings.TrimSpace(id) == "" {
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: approval ID is required",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked(time.Now().UTC())

	pending, found := m.pending[id]
	if !found {
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: approval was not found",
			ErrMCPInvalidRuntimeRequest,
		)
	}
	if pending.Issued || pending.Consumed {
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: approval was already resolved",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	result := MCPApprovalResolutionResult{
		ApprovalID: pending.ID,
		Resolution: resolution,
	}

	switch resolution {
	case MCPApprovalResolutionDenyOnce:
		delete(m.pending, id)
		result.Decision = MCPApprovalDecisionDenied
		return result, nil

	case MCPApprovalResolutionDenyAlways:
		if m.decisions == nil {
			m.decisions = make(map[string]rememberedApprovalDecision)
		}
		m.decisions[approvalDecisionKeyFor(pending.Summary)] = rememberedApprovalDecision{
			Summary:    cloneApprovalSummary(pending.Summary),
			Resolution: resolution,
		}
		delete(m.pending, id)
		result.Decision = MCPApprovalDecisionDenied
		result.RememberedForSession = true
		return result, nil

	case MCPApprovalResolutionAllowAlways:
		if m.decisions == nil {
			m.decisions = make(map[string]rememberedApprovalDecision)
		}
		m.decisions[approvalDecisionKeyFor(pending.Summary)] = rememberedApprovalDecision{
			Summary:    cloneApprovalSummary(pending.Summary),
			Resolution: resolution,
		}
		delete(m.pending, id)
		result.Decision = MCPApprovalDecisionAllowed
		result.RememberedForSession = true
		return result, nil

	case MCPApprovalResolutionAllowOnce:
		token, err := randomApprovalToken(32)
		if err != nil {
			return MCPApprovalResolutionResult{}, err
		}
		pending.Token = token
		pending.Issued = true

		result.Decision = MCPApprovalDecisionAllowed
		result.Token = token
		result.ExpiresAt = pending.ExpiresAt.UTC().Format(
			time.RFC3339Nano,
		)
		return result, nil

	default:
		return MCPApprovalResolutionResult{}, fmt.Errorf(
			"%w: unsupported approval resolution %q",
			ErrMCPInvalidRuntimeRequest,
			resolution,
		)
	}
}

func (m *ApprovalManager) LookupDecision(
	summary MCPApprovalSummary,
) (MCPApprovalResolution, bool) {
	if m == nil {
		return "", false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked(time.Now().UTC())
	decision, found := m.decisions[approvalDecisionKeyFor(summary)]
	if !found {
		return "", false
	}
	return decision.Resolution, true
}

// ClearServer removes pending tokens and remembered decisions belonging to an
// MCP server. It is called whenever that server's runtime session ends.
func (m *ApprovalManager) ClearServer(
	server mcpSpec.ServerID,
) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for id, pending := range m.pending {
		if pending.Summary.Server == server {
			delete(m.pending, id)
		}
	}
	for key, decision := range m.decisions {
		if decision.Summary.Server == server {
			delete(m.decisions, key)
		}
	}
}

func (m *ApprovalManager) Clear() {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.pending = make(map[string]*pendingApproval)
	m.decisions = make(map[string]rememberedApprovalDecision)
	m.mu.Unlock()
}

func (m *ApprovalManager) VerifyAndConsumeToken(
	ctx context.Context,
	token string,
	expected MCPApprovalSummary,
) (string, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return "", err
	}
	if m == nil {
		return "", fmt.Errorf(
			"%w: MCP approval manager is unavailable",
			ErrMCPRuntimeNotReady,
		)
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf(
			"%w: approval token is required",
			ErrMCPApprovalNeeded,
		)
	}
	if err := validateApprovalSummary(expected); err != nil {
		return "", err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked(time.Now().UTC())

	for _, pending := range m.pending {
		if pending.Token == "" {
			continue
		}
		if subtle.ConstantTimeCompare(
			[]byte(pending.Token),
			[]byte(token),
		) != 1 {
			continue
		}
		if pending.Consumed {
			return "", fmt.Errorf(
				"%w: approval token was already consumed",
				ErrMCPApprovalNeeded,
			)
		}
		if !approvalSummaryMatches(pending.Summary, expected) {
			return "", fmt.Errorf(
				"%w: approval token does not match this MCP tool call",
				ErrMCPApprovalNeeded,
			)
		}

		pending.Consumed = true
		delete(m.pending, pending.ID)
		return pending.ID, nil
	}

	return "", fmt.Errorf(
		"%w: approval token was not found",
		ErrMCPApprovalNeeded,
	)
}

func (m *ApprovalManager) purgeExpiredLocked(now time.Time) {
	for id, pending := range m.pending {
		if now.After(pending.ExpiresAt) {
			delete(m.pending, id)
		}
	}
}

func approvalDecisionKeyFor(summary MCPApprovalSummary) string {
	raw, _ := json.Marshal(approvalDecisionKey{
		Server:     summary.Server,
		ToolName:   summary.ToolName,
		ToolDigest: summary.ToolDigest,
		Risk:       summary.Risk,
		Source:     summary.Source,
		AppID:      summary.AppInstanceID,
	})
	return string(mcpSpec.DigestBytes(raw))
}

func approvalSummaryMatches(
	stored MCPApprovalSummary,
	expected MCPApprovalSummary,
) bool {
	if stored.Server != expected.Server ||
		stored.ToolName != expected.ToolName ||
		stored.Risk != expected.Risk ||
		stored.Source != expected.Source ||
		stored.AppInstanceID != expected.AppInstanceID {
		return false
	}
	if stored.ToolDigest != "" &&
		expected.ToolDigest != "" &&
		stored.ToolDigest != expected.ToolDigest {
		return false
	}

	return normalizeApprovalArguments(stored.Arguments) ==
		normalizeApprovalArguments(expected.Arguments)
}

func normalizeApprovalArguments(value jsonutil.JSONRawString) jsonutil.JSONRawString {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" {
		return jsonutil.JSONRawString(`{}`)
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return jsonutil.JSONRawString(trimmed)
	}

	normalized, err := json.Marshal(decoded)
	if err != nil {
		return jsonutil.JSONRawString(trimmed)
	}
	return jsonutil.JSONRawString(normalized)
}

func validateApprovalContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP approval context is nil",
			mcpSpec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func validateApprovalSummary(value MCPApprovalSummary) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := validateMCPInvocationSource(value.Source); err != nil {
		return err
	}
	if err := mcpSpec.ValidateOptionalText(
		"MCP approval app instance ID",
		value.AppInstanceID,
		mcpSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if value.Source == MCPInvocationSourceApp &&
		strings.TrimSpace(value.AppInstanceID) == "" {
		return fmt.Errorf(
			"%w: appInstanceID is required for an app approval",
			ErrMCPInvalidRuntimeRequest,
		)
	}
	if err := mcpSpec.ValidateRequiredText(
		"MCP approval tool name",
		value.ToolName,
		mcpSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}

	switch value.Risk {
	case MCPToolRiskUnknown,
		MCPToolRiskRead,
		MCPToolRiskWrite,
		MCPToolRiskDestructive,
		MCPToolRiskOpenWorld:
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval risk %q",
			mcpSpec.ErrInvalid,
			value.Risk,
		)
	}

	return nil
}

func cloneApprovalSummary(
	input MCPApprovalSummary,
) MCPApprovalSummary {
	output := input
	output.Arguments = jsonutil.JSONRawString(
		append([]byte(nil), []byte(input.Arguments)...),
	)
	return output
}

func randomApprovalToken(bytes int) (string, error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
