package logrotate

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Options define configuration options for Writer.
type Options struct {
	// Directory defines the directory where log files will be written to.
	// If the directory does not exist, it will be created.
	Directory string

	// MaximumFileSize defines the maximum size of each log file in bytes.
	// When MaximumFileSize == 0, no upper bound will be enforced.
	// No file will be greater than MaximumFileSize. A Write() which would
	// exceed MaximumFileSize will instead cause a new file to be created.
	// If a Write() is attempting to write more bytes than specified by
	// MaximumFileSize, the write will be skipped.
	MaximumFileSize int64

	// MaximumLifetime defines the maximum amount of time a file will
	// be written to before a rotation occurs.
	// When MaximumLifetime == 0, no log rotation will occur.
	MaximumLifetime time.Duration

	// FileNameFunc specifies the name a new file will take.
	// FileNameFunc must ensure collisions in filenames do not occur.
	// Do not rely on timestamps to be unique, high throughput writes
	// may fall on the same timestamp.
	// Eg.
	// 	2020-03-28_15-00-945-<random-hash>.log
	// When FileNameFunc is not specified, DefaultFilenameFunc will be used.
	FileNameFunc func() string

	// FlushAfterEveryWrite specifies whether the writer should flush
	// the buffer after every write.
	FlushAfterEveryWrite bool
}

// Writer is a concurrency-safe writer with file rotation.
type Writer struct {
	logger *slog.Logger

	// Opts are the configuration options for this Writer.
	opts Options

	// F is the currently open file used for appends.
	// Writes to f are only synchronized once Close() is called,
	// or when files are being rotated.
	f *os.File
	// Bw is a buffered writer for writing to f.
	bw *bufio.Writer
	// BytesWritten is the number of bytes written to f so far,
	// used for size based rotation.
	bytesWritten int64
	// Ts is the creation timestamp of f, used for time based log rotation.
	ts time.Time
	// Queue of entries awaiting to be written.
	queue chan []byte
	// Synchronize write which have started but not been queued up.
	pending sync.WaitGroup
	// Signal the writer should close.
	closing chan struct{}
	// Signal the writer has finished writing all queued up entries.
	done chan struct{}

	closeOnce sync.Once
	closeErr  error
}

// New creates a new concurrency safe Writer which performs log rotation.
func New(logger *slog.Logger, opts Options) (*Writer, error) {
	if _, err := os.Stat(opts.Directory); os.IsNotExist(err) {
		if err := os.MkdirAll(opts.Directory, os.ModePerm); err != nil {
			return nil, fmt.Errorf(
				"directory %v does not exist and could not be created, %w",
				opts.Directory,
				err,
			)
		}
	}

	if opts.FileNameFunc == nil {
		opts.FileNameFunc = DefaultFilenameFunc
	}

	w := &Writer{
		logger:  logger,
		opts:    opts,
		queue:   make(chan []byte, 4096),
		closing: make(chan struct{}),
		done:    make(chan struct{}),
	}

	go w.listen()

	return w, nil
}

func DefaultFilenameFunc() string {
	return fmt.Sprintf(
		"%s-%s.log",
		strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339), ":", "-"),
		RandomHash(3),
	)
}

// Write writes p into the current file, rotating if necessary.
// Write is non-blocking, if the writer's queue is not full.
// Write is blocking otherwise.
func (w *Writer) Write(p []byte) (n int, err error) {
	select {
	case <-w.closing:
		return 0, fmt.Errorf("writer is closing: %w", err)
	default:
		w.pending.Add(1)
		defer w.pending.Done()
	}
	copyP := make([]byte, len(p))
	copy(copyP, p)
	w.queue <- copyP

	return len(p), nil
}

// Close closes the writer.
// Any accepted writes will be flushed. Any new writes will be rejected.
// Once Close() exits, files are synchronized to disk.
func (w *Writer) Close() error {
	w.closeOnce.Do(func() {
		// Signal listen to stop accepting new writes.
		close(w.closing)
		// Wait for any pending Write() calls to finish adding to the queue.
		w.pending.Wait()
		// Close the queue so listen() drains and exits.
		close(w.queue)
		// Wait for listen() to finish processing and signal done.
		<-w.done
		if w.f != nil {
			w.closeErr = w.closeCurrentFile()
		}
	})
	return w.closeErr
}

func (w *Writer) listen() {
	for b := range w.queue {
		if w.f == nil {
			if err := w.rotate(); err != nil {
				w.logger.Error("failed to create log file", "error", err)
			}
		}

		size := int64(len(b))

		if w.opts.MaximumFileSize != 0 && size > w.opts.MaximumFileSize {
			w.logger.Info(
				"attempting to write more bytes than allowed by MaximumFileSize, skipping",
			)
			continue
		}
		if w.opts.MaximumFileSize != 0 && w.bytesWritten+size > w.opts.MaximumFileSize {
			if err := w.rotate(); err != nil {
				w.logger.Error("failed to rotate log file", "error", err)
			}
		}

		if w.opts.MaximumLifetime != 0 && time.Now().After(w.ts.Add(w.opts.MaximumLifetime)) {
			if err := w.rotate(); err != nil {
				w.logger.Error("failed to rotate log file", "error", err)
			}
		}

		if _, err := w.bw.Write(b); err != nil {
			w.logger.Error("failed to write to file", "error", err)
		}

		if w.opts.FlushAfterEveryWrite {
			if err := w.flushCurrentFile(); err != nil {
				w.logger.Error("failed to flush to file", "error", err)
			}
		}

		w.bytesWritten += size
	}

	close(w.done)
}

func (w *Writer) rotate() error {
	if w.f != nil {
		if err := w.closeCurrentFile(); err != nil {
			return err
		}
	}

	path := filepath.Join(w.opts.Directory, w.opts.FileNameFunc())
	f, err := newFile(path)
	if err != nil {
		return fmt.Errorf("failed to create new file at: %v, %w", path, err)
	}

	w.bw = bufio.NewWriter(f)
	w.f = f
	w.bytesWritten = 0
	w.ts = time.Now().UTC()

	return nil
}

func (w *Writer) closeCurrentFile() error {
	if err := w.flushCurrentFile(); err != nil {
		return err
	}

	// Close the underlying file so other processes (including RemoveAll on Windows)
	// can delete the file. Also clear writer references.
	if w.f != nil {
		if err := w.f.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}
		w.f = nil
	}
	w.bw = nil
	w.bytesWritten = 0

	return nil
}

func (w *Writer) flushCurrentFile() error {
	// If there is no open file or buffered writer, nothing to do.
	if w.bw == nil || w.f == nil {
		return nil
	}

	if err := w.bw.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffered writer: %w", err)
	}

	if err := w.f.Sync(); err != nil {
		return fmt.Errorf("failed to sync current log file: %w", err)
	}

	// Do NOT reset bytesWritten here — bytesWritten represents the total bytes
	// written to the current file and must only be reset when a new file is created.
	return nil
}

func newFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o666)
}
