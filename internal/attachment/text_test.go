package attachment

import "testing"

func TestFormatTextBlockForLLM_PrefersFilePathOverFileName(t *testing.T) {
	text := "package main\n"
	fileName := "main.go"
	filePath := "/repo/app/main.go"

	got, err := FormatTextBlockForLLM(ContentBlock{
		Kind:     ContentBlockText,
		Text:     &text,
		FileName: &fileName,
		FilePath: &filePath,
	})
	if err != nil {
		t.Fatalf("FormatTextBlockForLLM returned error: %v", err)
	}

	want := "<<<FILE path=\"/repo/app/main.go\">>>\npackage main\n\n<<<END FILE>>>"
	if got != want {
		t.Fatalf("unexpected formatted text block\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFormatTextBlockForLLM_FallsBackToFileName(t *testing.T) {
	text := "hello"
	fileName := "note.txt"

	got, err := FormatTextBlockForLLM(ContentBlock{
		Kind:     ContentBlockText,
		Text:     &text,
		FileName: &fileName,
	})
	if err != nil {
		t.Fatalf("FormatTextBlockForLLM returned error: %v", err)
	}

	want := "<<<FILE name=\"note.txt\">>>\nhello\n\n<<<END FILE>>>"
	if got != want {
		t.Fatalf("unexpected formatted text block\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestGetTextBlockWithDisplayNameOnly_LocalFileCarriesFileMetadata(t *testing.T) {
	att := &Attachment{
		Kind:  AttachmentFile,
		Label: "main.go",
		FileRef: &FileRef{
			PathInfo: PathInfo{
				Path: "/repo/app/main.go",
				Name: "main.go",
			},
		},
	}

	cb, err := att.GetTextBlockWithDisplayNameOnly("(binary file; not readable in this chat)")
	if err != nil {
		t.Fatalf("GetTextBlockWithDisplayNameOnly returned error: %v", err)
	}
	if cb == nil {
		t.Fatal("expected non-nil content block")
	}
	if cb.FilePath == nil || *cb.FilePath != "/repo/app/main.go" {
		t.Fatalf("expected file path metadata to be carried, got %#v", cb.FilePath)
	}
	if cb.FileName == nil || *cb.FileName != "main.go" {
		t.Fatalf("expected file name metadata to be carried, got %#v", cb.FileName)
	}
}
