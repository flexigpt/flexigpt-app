package attachment

import "strings"

// FileFilter is used to build file dialog filters (Patterns like "*.png;*.jpg").
type FileFilter struct {
	DisplayName string
	Extensions  []FileExt // normalized, e.g. ".png".
}

var (
	FileFilterAllFiles = FileFilter{
		DisplayName: "All Files",
		Extensions:  nil, // Pattern() => "*".
	}

	FileFilterTextMarkdown = FileFilter{
		DisplayName: "Text",
		Extensions:  ModeToExtensions[ExtensionModeText],
	}

	FileFilterDocuments = FileFilter{
		DisplayName: "Documents",
		Extensions:  ModeToExtensions[ExtensionModeDocument],
	}

	FileFilterImages = FileFilter{
		DisplayName: "Images",
		Extensions:  ModeToExtensions[ExtensionModeImage],
	}

	DefaultFileFilters = []FileFilter{
		FileFilterAllFiles,
		FileFilterTextMarkdown,
		FileFilterDocuments,
		FileFilterImages,
	}
)

// Pattern converts a FileFilter into a semicolon-separated pattern string.
// W.g. "*.png;*.jpg".
func (f FileFilter) Pattern() string {
	if len(f.Extensions) == 0 {
		return "*"
	}
	out := make([]string, 0, len(f.Extensions))
	for _, ext := range f.Extensions {
		e := strings.TrimSpace(string(ext))
		// Be liberal in what we accept from callers/frontends: allow ".txt", "txt", "*.txt", "*txt".
		e = strings.TrimPrefix(e, "*")
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, "*"+e)
	}
	if len(out) == 0 {
		return "*"
	}
	return strings.Join(out, ";")
}
