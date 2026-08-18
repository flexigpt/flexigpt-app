package jsonutil

import (
	"context"
	"encoding/json"
	"log/slog"
)

func LogJSON(level slog.Level, v any) {
	p, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		slog.Log(context.Background(), level, "params", "json", string(p))
	}
}
