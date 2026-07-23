package skillstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

const embeddedHydrateDigestFile = ".embeddedfs.sha256"

func (s *SkillStore) materializeBuiltInEmbeddedFS(
	ctx context.Context,
) (err error) {
	if s == nil || s.builtin == nil || s.builtin.skillsFS == nil {
		return nil
	}

	s.embeddedMaterializeMu.Lock()
	defer s.embeddedMaterializeMu.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("materialize built-in Skills: panic", "panic", recovered)
			err = fmt.Errorf("materialize built-in Skills panic: %v", recovered)
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
	destination := s.embeddedHydrateDir
	if strings.TrimSpace(destination) == "" {
		return errors.New("embedded Skill hydration directory is empty")
	}

	digestPath := filepath.Join(destination, embeddedHydrateDigestFile)
	previous, _ := os.ReadFile(digestPath)
	if strings.TrimSpace(string(previous)) == digest {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temporary := fmt.Sprintf("%s.tmp-%d", destination, time.Now().UnixNano())
	_ = os.RemoveAll(temporary)
	if err := os.MkdirAll(temporary, 0o755); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temporary) }()
	if err := copyFSToDir(sub, temporary); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(temporary, embeddedHydrateDigestFile),
		[]byte(digest+"\n"),
		0o600,
	); err != nil {
		return err
	}

	previousDir := ""
	if _, err := os.Stat(destination); err == nil {
		previousDir = fmt.Sprintf("%s.old-%d", destination, time.Now().UnixNano())
		if err := os.Rename(destination, previousDir); err != nil {
			return err
		}
	}
	if err := os.Rename(temporary, destination); err != nil {
		if previousDir != "" {
			_ = os.Rename(previousDir, destination)
		}
		return err
	}
	if previousDir != "" {
		_ = os.RemoveAll(previousDir)
	}
	slog.Info("hydrated embedded skills fs", "dir", destination, "digest", digest)
	return nil
}

func fsDigestSHA256(fsys fs.FS) (string, error) {
	hash := sha256.New()
	var paths []string
	if err := fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	for _, path := range paths {
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hash, path)
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(content)
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyFSToDir(fsys fs.FS, destination string) error {
	return fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outputPath := filepath.Join(destination, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(outputPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		permission := info.Mode().Perm()
		if permission == 0 {
			permission = 0o644
		}
		input, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, permission)
		if err != nil {
			return err
		}
		if _, err := io.Copy(output, input); err != nil {
			_ = output.Close()
			return err
		}
		return output.Close()
	})
}
