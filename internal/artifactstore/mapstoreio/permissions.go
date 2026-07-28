package mapstoreio

import "io/fs"

func ApplyPrivateDirectoryMode(location string) error {
	return applyPrivateMode(location, fs.FileMode(PrivateDirectoryMode))
}

func ApplyPrivateFileMode(location string) error {
	return applyPrivateMode(location, fs.FileMode(PrivateFileMode))
}
