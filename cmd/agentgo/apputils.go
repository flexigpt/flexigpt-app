package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
)

const FrontendPathPrefix = "/frontend/dist"

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

func URLCleanerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Clean the URL using getActualURL.
		cleanedURL := getActualURL(req.URL.Path)

		// Update the request URL path.
		req.URL.Path = cleanedURL

		// Call the next handler.
		next.ServeHTTP(w, req)
	})
}

func getActualURL(origurl string) string {
	callurl := origurl
	if !strings.HasPrefix(callurl, FrontendPathPrefix) {
		return callurl
	}

	// Handle if it's a page request.
	if strings.HasSuffix(callurl, "/") {
		return callurl[len(FrontendPathPrefix):] + "index.html"
	}

	for _, d := range DIRPages {
		durl := FrontendPathPrefix + d
		if callurl == durl {
			return callurl[len(FrontendPathPrefix):] + "/index.html"
		}
	}

	return callurl[len(FrontendPathPrefix):]
}
