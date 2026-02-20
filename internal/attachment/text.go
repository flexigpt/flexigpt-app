package attachment

import (
	"strings"
)

func FormatTextBlockForLLM(b ContentBlock) (string, error) {
	if b.FileName == nil && b.URL == nil && b.Text == nil {
		return "", nil
	}

	var sb strings.Builder

	sb.WriteString("<<<FILE")
	if b.FileName != nil {
		sb.WriteString(` name="`)
		sb.WriteString(*b.FileName) // no escaping
		sb.WriteString(`"`)
	}
	if b.URL != nil {
		sb.WriteString(` url="`)
		sb.WriteString(*b.URL) // no escaping
		sb.WriteString(`"`)
	}
	sb.WriteString(">>>\n")

	text := ""
	if b.Text != nil {
		text = *b.Text // raw
	}

	sb.WriteString(text)
	if text == "" || text[len(text)-1] != '\n' {
		sb.WriteByte('\n')
	}

	sb.WriteByte('\n')

	sb.WriteString("<<<END FILE>>>")

	return sb.String(), nil
}
