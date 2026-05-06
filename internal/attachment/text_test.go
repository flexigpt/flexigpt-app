package attachment

import "testing"

const (
	testMainGoFileName = "main.go"
	testMainGoFilePath = "/repo/app/main.go"
	testNoteFileName   = "note.txt"
)

func TestFormatTextBlockForLLM_PrefersFilePathOverFileName(t *testing.T) {
	text := "package main\n"
	fileName := testMainGoFileName
	filePath := testMainGoFilePath

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
	fileName := testNoteFileName

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
				Path: testMainGoFilePath,
				Name: testMainGoFileName,
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
	if cb.FilePath == nil || *cb.FilePath != testMainGoFilePath {
		t.Fatalf("expected file path metadata to be carried, got %#v", cb.FilePath)
	}
	if cb.FileName == nil || *cb.FileName != testMainGoFileName {
		t.Fatalf("expected file name metadata to be carried, got %#v", cb.FileName)
	}
}
