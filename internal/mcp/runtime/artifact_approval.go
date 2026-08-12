//nolint:nilnil // Required.
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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const defaultApprovalTTL = 5 * time.Minute

type approvalDecisionKey struct {
	Server     artifact.ArtifactRef `json:"server"`
	ToolName   string               `json:"toolName"`
	ToolDigest string               `json:"toolDigest,omitempty"`
	Risk       MCPToolRisk          `json:"risk"`
}

type pendingApproval struct {
	ID        string
	Token     string
	Summary   MCPApprovalSummary
	ExpiresAt time.Time
	Issued    bool
	Consumed  bool
}

type ApprovalManager struct {
	mu        sync.Mutex
	ttl       time.Duration
	pending   map[string]*pendingApproval
	decisions map[string]MCPApprovalResolution
}

func NewApprovalManager(ttl time.Duration) *ApprovalManager {
	if ttl <= 0 {
		ttl = defaultApprovalTTL
	}
	return &ApprovalManager{
		ttl:       ttl,
		pending:   make(map[string]*pendingApproval),
		decisions: make(map[string]MCPApprovalResolution),
	}
}

func (m *ApprovalManager) Create(
	ctx context.Context,
	summary MCPApprovalSummary,
) (string, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return "", err
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
) (*MCPApprovalToken, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf(
			"%w: approval ID is required",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.purgeExpiredLocked(time.Now().UTC())

	pending, found := m.pending[id]
	if !found {
		return nil, fmt.Errorf(
			"%w: approval was not found",
			ErrMCPInvalidRuntimeRequest,
		)
	}
	if pending.Issued || pending.Consumed {
		return nil, fmt.Errorf(
			"%w: approval was already resolved",
			ErrMCPInvalidRuntimeRequest,
		)
	}

	switch resolution {
	case MCPApprovalResolutionDenyOnce:
		delete(m.pending, id)
		return nil, nil

	case MCPApprovalResolutionDenyAlways:
		m.decisions[approvalDecisionKeyFor(pending.Summary)] = resolution
		delete(m.pending, id)
		return nil, nil

	case MCPApprovalResolutionAllowAlways:
		m.decisions[approvalDecisionKeyFor(pending.Summary)] = resolution
		delete(m.pending, id)
		return nil, nil

	case MCPApprovalResolutionAllowOnce:
		token, err := randomApprovalToken(32)
		if err != nil {
			return nil, err
		}
		pending.Token = token
		pending.Issued = true

		return &MCPApprovalToken{
			ApprovalID: pending.ID,
			Token:      token,
			ExpiresAt:  pending.ExpiresAt.UTC().Format(time.RFC3339Nano),
		}, nil

	default:
		return nil, fmt.Errorf(
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
	value, found := m.decisions[approvalDecisionKeyFor(summary)]
	return value, found
}

func (m *ApprovalManager) VerifyAndConsumeToken(
	ctx context.Context,
	token string,
	expected MCPApprovalSummary,
) (string, error) {
	if err := validateApprovalContext(ctx); err != nil {
		return "", err
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
	})
	return string(cryptoutil.DigestBytes(raw))
}

func approvalSummaryMatches(
	stored MCPApprovalSummary,
	expected MCPApprovalSummary,
) bool {
	if stored.Server != expected.Server ||
		stored.ToolName != expected.ToolName ||
		stored.Risk != expected.Risk {
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
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return jsonutil.JSONRawString(`{}`)
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}

	normalized, err := json.Marshal(decoded)
	if err != nil {
		return trimmed
	}
	return jsonutil.JSONRawString(normalized)
}

func validateApprovalContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP approval context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}

func validateApprovalSummary(value MCPApprovalSummary) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP approval tool name",
		value.ToolName,
		basespec.MaxDisplayNameBytes,
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
			basespec.ErrInvalid,
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
