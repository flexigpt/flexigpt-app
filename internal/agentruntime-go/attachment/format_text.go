package attachment

import (
	"strings"
)

func FormatTextBlockForLLM(b ContentBlock) (string, error) {
	filePath := ""
	if b.FilePath != nil {
		filePath = strings.TrimSpace(*b.FilePath)
	}

	fileName := ""
	if b.FileName != nil {
		fileName = strings.TrimSpace(*b.FileName)
	}

	blockURL := ""
	if b.URL != nil {
		blockURL = strings.TrimSpace(*b.URL)
	}

	if filePath == "" && fileName == "" && blockURL == "" && b.Text == nil {
		return "", nil
	}

	var sb strings.Builder

	sb.WriteString("<<<FILE")
	switch {
	case filePath != "":
		sb.WriteString(` path="`)
		sb.WriteString(filePath) // no escaping
		sb.WriteString(`"`)
	case fileName != "":
		sb.WriteString(` name="`)
		sb.WriteString(fileName) // no escaping
		sb.WriteString(`"`)
	}
	if blockURL != "" {
		sb.WriteString(` url="`)
		sb.WriteString(blockURL) // no escaping
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
