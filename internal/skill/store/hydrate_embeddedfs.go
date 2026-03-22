package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/fsutil"
)

func (s *SkillStore) hydrateBuiltInEmbeddedFS(ctx context.Context) (err error) {
	if s.builtin == nil {
		return nil
	}

	s.embeddedHydrateMu.Lock()
	defer s.embeddedHydrateMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("hydrateBuiltInEmbeddedFS: panic", "panic", r)
			err = fmt.Errorf("hydrateBuiltInEmbeddedFS panic: %v", r)

		}
	}()

	sub, err := fsutil.ResolveFS(s.builtin.skillsFS, s.builtin.skillsDir)
	if err != nil {
		return err
	}
	digest, err := fsDigestSHA256(sub)
	if err != nil {
		return err
	}

	digestPath := filepath.Join(s.embeddedHydrateDir, embeddedHydrateDigestFile)
	prev, _ := os.ReadFile(digestPath)
	if strings.TrimSpace(string(prev)) == digest {
		return nil
	}

	parent := filepath.Dir(s.embeddedHydrateDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	tmpDir := fmt.Sprintf("%s.tmp-%d", s.embeddedHydrateDir, time.Now().UnixNano())
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := copyFSToDir(sub, tmpDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, embeddedHydrateDigestFile), []byte(digest+"\n"), 0o600); err != nil {
		return err
	}

	// Swap atomically-ish: rename existing aside, then rename tmp into place.
	oldDir := ""
	if _, err := os.Stat(s.embeddedHydrateDir); err == nil {
		oldDir = fmt.Sprintf("%s.old-%d", s.embeddedHydrateDir, time.Now().UnixNano())
		if err := os.Rename(s.embeddedHydrateDir, oldDir); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpDir, s.embeddedHydrateDir); err != nil {
		// Best-effort rollback.
		if oldDir != "" {
			_ = os.Rename(oldDir, s.embeddedHydrateDir)
		}
		return err
	}
	if oldDir != "" {
		_ = os.RemoveAll(oldDir)
	}

	slog.Info("hydrated embedded skills fs", "dir", s.embeddedHydrateDir, "digest", digest)
	return nil
}

func fsDigestSHA256(fsys fs.FS) (string, error) {
	h := sha256.New()
	var paths []string
	if err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, p)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)
	for _, p := range paths {
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFSToDir(fsys fs.FS, dest string) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outPath := filepath.Join(dest, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		perm := fs.FileMode(0o644)
		if m := info.Mode().Perm(); m != 0 {
			perm = m
		}

		in, err := fsys.Open(p)
		if err != nil {
			return err
		}

		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
		if err != nil {
			_ = in.Close()
			return err
		}

		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			_ = in.Close()
			return err
		}
		_ = out.Close()
		_ = in.Close()
		return nil
	})
}
