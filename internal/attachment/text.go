package attachment

import (
	"encoding/xml"
	"strings"
)

type llmTextAttachment struct {
	XMLName    xml.Name `xml:"attachment"`
	SourceFile string   `xml:"sourceFile,attr,omitempty"` // optional
	SourceURL  string   `xml:"sourceURL,attr,omitempty"`  // optional
	Text       string   `xml:",chardata"`                 // escaped automatically
}

// XML 1.0 disallows some control chars; this prevents xml.Marshal errors on odd file bytes.
func sanitizeXMLText(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return r
		case r >= 0x20:
			return r
		default:
			return -1
		}
	}, s)
}

func FormatTextBlockAsXML(b ContentBlock) (string, error) {
	var out llmTextAttachment
	if b.FileName == nil && b.URL == nil && b.Text == nil {
		return "", nil
	}

	if b.FileName != nil {
		out.SourceFile = strings.TrimSpace(*b.FileName)
	}
	if b.URL != nil {
		out.SourceURL = strings.TrimSpace(*b.URL)
	}

	if b.Text != nil {
		// You can decide whether you want TrimSpace here; trimming can change file contents.
		out.Text = sanitizeXMLText(*b.Text)
	}

	blob, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(blob), nil
}
