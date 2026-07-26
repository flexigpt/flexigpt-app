package mapstoreio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PrivateDirectoryMode = 0o700
	PrivateFileMode      = 0o600
)

func PreparePrivateDirectory(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("private directory path is empty")
	}
	absolute, err := filepath.Abs(raw)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)

	if err := os.MkdirAll(absolute, PrivateDirectoryMode); err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf(
			"private path %q is not a non-symlink directory",
			absolute,
		)
	}
	if err := os.Chmod(absolute, PrivateDirectoryMode); err != nil {
		return "", err
	}
	return absolute, nil
}

func EnsurePrivateSubdirectory(
	base string,
	relative string,
) (string, error) {
	return privateSubdirectory(base, relative, true)
}

// PrivateSubdirectoryPath returns a validated path beneath a private base
// without creating missing child directories. Existing components are checked
// with Lstat so a symlink cannot redirect managed content outside its root.
func PrivateSubdirectoryPath(
	base string,
	relative string,
) (string, error) {
	return privateSubdirectory(base, relative, false)
}

func privateSubdirectory(
	base string,
	relative string,
	create bool,
) (string, error) {
	base, err := PreparePrivateDirectory(base)
	if err != nil {
		return "", err
	}
	clean, err := normalizePrivateRelativePath(relative)
	if err != nil {
		return "", err
	}
	if clean == "" {
		return base, nil
	}

	current := base
	for segment := range strings.SplitSeq(clean, string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		switch {
		case statErr == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return "", fmt.Errorf(
					"private path component %q is not a non-symlink directory",
					current,
				)
			}
		case errors.Is(statErr, os.ErrNotExist):
			if !create {
				return filepath.Join(base, clean), nil
			}
			if err := os.Mkdir(current, PrivateDirectoryMode); err != nil &&
				!errors.Is(err, os.ErrExist) {
				return "", err
			}
			info, statErr = os.Lstat(current)
			if statErr != nil {
				return "", statErr
			}
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return "", fmt.Errorf(
					"private path component %q is not a non-symlink directory",
					current,
				)
			}
		default:
			return "", statErr
		}
		if err := os.Chmod(current, PrivateDirectoryMode); err != nil {
			return "", err
		}
	}
	if err := ensurePrivateContained(base, current); err != nil {
		return "", err
	}
	return current, nil
}

func normalizePrivateRelativePath(relative string) (string, error) {
	if relative == "" || relative == "." {
		return "", nil
	}
	if filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return "", errors.New("private subdirectory must be relative")
	}
	clean := filepath.Clean(relative)
	if clean == "." {
		return "", nil
	}
	if clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("private subdirectory escapes its base")
	}
	return clean, nil
}

func ensurePrivateContained(base, location string) error {
	relative, err := filepath.Rel(base, location)
	if err != nil {
		return err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return errors.New("private path escapes its base")
	}
	return nil
}

func PrivateFilePath(
	base string,
	partition string,
	fileName string,
	createParent bool,
) (string, error) {
	if fileName == "" ||
		fileName == "." ||
		fileName == ".." ||
		strings.ContainsAny(fileName, `/\`) ||
		filepath.VolumeName(fileName) != "" {
		return "", fmt.Errorf("invalid private filename %q", fileName)
	}

	base, err := PreparePrivateDirectory(base)
	if err != nil {
		return "", err
	}
	parent, err := privateSubdirectory(base, partition, createParent)
	if err != nil {
		return "", err
	}

	location := filepath.Clean(filepath.Join(parent, fileName))
	relative, err := filepath.Rel(base, location)
	if err != nil {
		return "", err
	}
	if relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relative) {
		return "", errors.New("private file path escapes its base")
	}
	return location, nil
}

func SecureRegularFile(location string) error {
	info, err := os.Lstat(location)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf(
			"private file %q is not a regular non-symlink file",
			location,
		)
	}
	return os.Chmod(location, PrivateFileMode)
}

func SyncRegularFile(location string) error {
	file, err := os.Open(location)
	if err != nil {
		return err
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	return errors.Join(syncErr, closeErr)
}
