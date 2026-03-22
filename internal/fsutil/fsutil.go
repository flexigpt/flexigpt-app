package fsutil

import (
	"fmt"
	"io/fs"
)

func ResolveFS(fsys fs.FS, dir string) (fs.FS, error) {
	if dir == "" || dir == "." {
		return fsys, nil
	}

	// Validate dir exists and is a directory.
	fi, err := fs.Stat(fsys, dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}

	return fs.Sub(fsys, dir)
}
