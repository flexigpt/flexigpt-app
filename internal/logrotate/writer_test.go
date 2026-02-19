package logrotate

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCreateTargetDirectory(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))
	dir, cleanup := setup(t)
	defer cleanup()

	dir = filepath.Join(dir, "foo")
	w, err := New(logger, Options{
		Directory: dir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	// Ensure writer is closed even on early test failure.
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("failed to close writer: %v", err)
		}
	})

	f, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}
	if !f.IsDir() {
		t.Fatalf("expected a directory to be created")
	}
}

func TestCreateWriteClose(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))

	dir, cleanup := setup(t)
	defer cleanup()

	w, err := New(logger, Options{
		Directory: dir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	// Ensure writer is closed even on early test failure.
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("failed to close writer: %v", err)
		}
	})

	message := []byte("message")
	if _, err := w.Write(message); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	written, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(written, message) {
		t.Fatalf("expected %s, got %s", message, written)
	}
}

func TestRotateOnFileSize(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))

	dir, cleanup := setup(t)
	defer cleanup()

	tests := []struct {
		maxSize  int
		writes   []string
		expFiles int
	}{
		{128, []string{strings.Repeat("a", 128), "b"}, 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("maxSize=%d", tt.maxSize), func(t *testing.T) {
			w, err := New(logger, Options{
				Directory:       dir,
				MaximumFileSize: int64(tt.maxSize),
			})
			if err != nil {
				t.Fatalf("failed to create writer: %v", err)
			}
			// Ensure writer is closed even on early test failure.
			t.Cleanup(func() {
				if err := w.Close(); err != nil {
					t.Errorf("failed to close writer: %v", err)
				}
			})

			for _, write := range tt.writes {
				if _, err := w.Write([]byte(write)); err != nil {
					t.Fatalf("failed to write: %v", err)
				}
			}

			if err := w.Close(); err != nil {
				t.Fatalf("failed to close writer: %v", err)
			}

			files, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("failed to read directory: %v", err)
			}

			if len(files) != tt.expFiles {
				t.Fatalf("expected %d files, got %d", tt.expFiles, len(files))
			}
		})
	}
}

func TestRotateOnLifetime(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))

	dir, cleanup := setup(t)
	defer cleanup()

	lifetime := time.Second
	w, err := New(logger, Options{
		Directory:       dir,
		MaximumLifetime: lifetime,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	// Ensure writer is closed even on early test failure.
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("failed to close writer: %v", err)
		}
	})

	// Keep writing until lifetime + half of lifetime (middle of ticks) elapses.
	end := time.Now().Add(lifetime + lifetime/2)
	for time.Now().Before(end) {
		if _, err := w.Write([]byte("message")); err != nil {
			t.Fatalf("failed to write: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestConcurrentWrites(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))

	dir, cleanup := setup(t)
	defer cleanup()

	w, err := New(logger, Options{
		Directory: dir,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	// Ensure writer is closed even on early test failure.
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("failed to close writer: %v", err)
		}
	})

	rows := 10000
	writers := 10
	messageSize := 10

	var wg sync.WaitGroup
	errCh := make(chan error, writers)

	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for range rows {
				if _, err := w.Write([]byte(strings.Repeat(strconv.Itoa(i), messageSize))); err != nil {
					errCh <- fmt.Errorf("failed to write: %w", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Error(err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	fileinfo, err := files[0].Info()
	if err != nil {
		t.Fatalf("failed to get file info: %v", err)
	}
	expectedSize := int64(rows * writers * messageSize)
	if fileinfo.Size() != expectedSize {
		t.Fatalf("expected file size %d, got %d", expectedSize, fileinfo.Size())
	}
}

func TestFlushAfterEveryWrite(t *testing.T) {
	slogOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, slogOpts))

	dir, cleanup := setup(t)
	defer cleanup()

	w, err := New(logger, Options{
		Directory:            dir,
		FlushAfterEveryWrite: true,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}
	// Ensure writer is closed even on early test failure.
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("failed to close writer: %v", err)
		}
	})

	if _, err := w.Write([]byte("message")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// The write is asynchronous, so we need to wait a bit.
	time.Sleep(1 * time.Second)

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Read the file.
	written, err := os.ReadFile(path.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(written) != "message" {
		t.Fatalf("expected 'message', got %s", written)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
}

func setup(t *testing.T) (dir string, cleanup func()) {
	t.Helper()
	tdir := t.TempDir()

	cleanup = func() {
	}

	return tdir, cleanup
}
