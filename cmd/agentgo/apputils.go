package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"runtime"
)

var DIRPages = []string{"/404"}

func EmbeddedFSWalker(assets embed.FS) {
	_ = fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		slog.Info("embedded walk", "path", path)
		return nil
	})
}

func LogStackTrace() {
	// Create a buffer to hold the stack trace.
	buf := make([]byte, 0, 4096)
	// Capture the stack trace.
	n := runtime.Stack(buf, false)
	// Log the stack trace.
	slog.Info("stack", "trace", string(buf[:n]))
}
