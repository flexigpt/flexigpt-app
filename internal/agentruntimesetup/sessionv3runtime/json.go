package sessionv3runtime

import (
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func canonicalValue(
	value any,
) (sessionv3Runtime.CanonicalValue, error) {
	raw, err := jsonutil.EncodeToJSONRaw(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical runtime value: %w", err)
	}

	canonical, err := sessionv3Runtime.CanonicalValueFrom(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize runtime value: %w", err)
	}

	return canonical, nil
}
