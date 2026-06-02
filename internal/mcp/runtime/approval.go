package runtime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/mapstore-go/uuidv7filename"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const defaultApprovalTTL = 5 * time.Minute

type approvalDecisionKey struct {
	BundleID   bundleitemutils.BundleID `json:"bundleID"`
	ServerID   spec.MCPServerID         `json:"serverID"`
	ToolName   string                   `json:"toolName"`
	ToolDigest string                   `json:"toolDigest,omitempty"`
	Risk       spec.MCPToolRisk         `json:"risk"`
	Arguments  spec.JSONRawString       `json:"arguments,omitempty"`
}

type pendingApproval struct {
	ID        string
	Token     string
	Summary   spec.MCPApprovalSummary
	ExpiresAt time.Time
	Consumed  bool
}

type ApprovalManager struct {
	mu        sync.Mutex
	ttl       time.Duration
	pending   map[string]*pendingApproval
	decisions map[string]spec.MCPApprovalResolution
}

func NewApprovalManager(ttl time.Duration) *ApprovalManager {
	if ttl <= 0 {
		ttl = defaultApprovalTTL
	}
	return &ApprovalManager{
		ttl:       ttl,
		pending:   map[string]*pendingApproval{},
		decisions: map[string]spec.MCPApprovalResolution{},
	}
}

func (m *ApprovalManager) Create(ctx context.Context, summary spec.MCPApprovalSummary) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return "", err
	}

	m.pending[id] = &pendingApproval{
		ID:        id,
		Summary:   summary,
		ExpiresAt: time.Now().UTC().Add(m.ttl),
	}
	return id, nil
}

func (m *ApprovalManager) Resolve(
	ctx context.Context,
	id string,
	res spec.MCPApprovalResolution,
) (*spec.MCPApprovalToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.pending[id]
	if !ok {
		return nil, fmt.Errorf("%w: approval not found", spec.ErrMCPInvalidRequest)
	}
	if time.Now().UTC().After(p.ExpiresAt) {
		delete(m.pending, id)
		return nil, fmt.Errorf("%w: approval expired", spec.ErrMCPInvalidRequest)
	}

	switch res {
	case spec.MCPApprovalResolutionDenyOnce, spec.MCPApprovalResolutionDenyAlways:
		delete(m.pending, id)
		if res == spec.MCPApprovalResolutionDenyAlways {
			m.decisions[getApprovalDecisionKey(p.Summary)] = res
		}
		//nolint:nilnil // Deny resolution is a successful resolution without a token.
		return nil, nil

	case spec.MCPApprovalResolutionAllowOnce, spec.MCPApprovalResolutionAllowAlways:
		token, err := randomToken()
		if err != nil {
			return nil, err
		}
		p.Token = token
		if res == spec.MCPApprovalResolutionAllowAlways {
			m.decisions[getApprovalDecisionKey(p.Summary)] = res
			delete(m.pending, id)
		}
		return &spec.MCPApprovalToken{
			ApprovalID: id,
			Token:      token,
			ExpiresAt:  p.ExpiresAt.Format(time.RFC3339Nano),
		}, nil

	default:
		return nil, fmt.Errorf("%w: invalid resolution", spec.ErrMCPInvalidRequest)
	}
}

func (m *ApprovalManager) LookupDecision(summary spec.MCPApprovalSummary) (spec.MCPApprovalResolution, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.decisions[getApprovalDecisionKey(summary)]
	return res, ok
}

func (m *ApprovalManager) VerifyAndConsume(ctx context.Context, id, token string) error {
	if id == "" || token == "" {
		return fmt.Errorf("%w: approval token required", spec.ErrMCPApprovalNeeded)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.pending[id]
	if !ok {
		return fmt.Errorf("%w: approval not found", spec.ErrMCPApprovalNeeded)
	}
	return m.verifyLocked(p, token, nil)
}

func (m *ApprovalManager) VerifyAndConsumeToken(
	ctx context.Context,
	token string,
	expected spec.MCPApprovalSummary,
) (string, error) {
	if token == "" {
		return "", fmt.Errorf("%w: approval token required", spec.ErrMCPApprovalNeeded)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.pending {
		if p.Token == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(p.Token), []byte(token)) != 1 {
			continue
		}
		id := p.ID
		return id, m.verifyLocked(p, token, &expected)
	}

	return "", fmt.Errorf("%w: approval not found", spec.ErrMCPApprovalNeeded)
}

func (m *ApprovalManager) verifyLocked(
	p *pendingApproval,
	token string,
	expected *spec.MCPApprovalSummary,
) error {
	if p == nil {
		return fmt.Errorf("%w: approval not found", spec.ErrMCPApprovalNeeded)
	}
	if p.Consumed {
		return fmt.Errorf("%w: approval already consumed", spec.ErrMCPApprovalNeeded)
	}
	if time.Now().UTC().After(p.ExpiresAt) {
		delete(m.pending, p.ID)
		return fmt.Errorf("%w: approval expired", spec.ErrMCPApprovalNeeded)
	}
	if p.Token == "" || subtle.ConstantTimeCompare([]byte(p.Token), []byte(token)) != 1 {
		return fmt.Errorf("%w: bad approval token", spec.ErrMCPApprovalNeeded)
	}
	if expected != nil && !summaryMatches(p.Summary, *expected) {
		return fmt.Errorf("%w: approval token does not match requested tool call", spec.ErrMCPApprovalNeeded)
	}

	p.Consumed = true
	delete(m.pending, p.ID)
	return nil
}

func getApprovalDecisionKey(summary spec.MCPApprovalSummary) string {
	key := approvalDecisionKey{
		BundleID:   summary.BundleID,
		ServerID:   summary.ServerID,
		ToolName:   summary.ToolName,
		ToolDigest: summary.ToolDigest,
		Risk:       summary.Risk,
		Arguments:  normalizeRawJSON(summary.Arguments),
	}
	raw, _ := json.Marshal(key)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func summaryMatches(stored, expected spec.MCPApprovalSummary) bool {
	if stored.BundleID != expected.BundleID {
		return false
	}
	if stored.ServerID != expected.ServerID {
		return false
	}
	if stored.ToolName != expected.ToolName {
		return false
	}
	if stored.Risk != expected.Risk {
		return false
	}
	if stored.ToolDigest != "" && expected.ToolDigest != "" && stored.ToolDigest != expected.ToolDigest {
		return false
	}
	if stored.Arguments != "" && expected.Arguments != "" &&
		normalizeRawJSON(stored.Arguments) != normalizeRawJSON(expected.Arguments) {
		return false
	}
	return true
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func normalizeRawJSON(s spec.JSONRawString) spec.JSONRawString {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return spec.JSONRawString(`{}`)
	}

	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return trimmed
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return trimmed
	}
	return spec.JSONRawString(raw)
}
