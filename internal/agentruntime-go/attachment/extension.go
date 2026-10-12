package attachment

type ExtensionMode string

const (
	ExtensionModeText     ExtensionMode = "text"
	ExtensionModeImage    ExtensionMode = "image"
	ExtensionModeDocument ExtensionMode = "document"
	ExtensionModeDefault  ExtensionMode = "default"
)

var AllExtensionModes = []ExtensionMode{
	ExtensionModeText,
	ExtensionModeImage,
	ExtensionModeDocument,
	ExtensionModeDefault,
}

type FileExt string

const (
	ExtTxt      FileExt = ".txt"
	ExtMd       FileExt = ".md"
	ExtMarkdown FileExt = ".markdown"
	ExtMdx      FileExt = ".mdx"
	ExtLog      FileExt = ".log"
	ExtJSON     FileExt = ".json"
	ExtYAML     FileExt = ".yaml"
	ExtYML      FileExt = ".yml"
	ExtTOML     FileExt = ".toml"
	ExtJS       FileExt = ".js"
	ExtTS       FileExt = ".ts"
	ExtTSX      FileExt = ".tsx"
	ExtJSX      FileExt = ".jsx"
	ExtPY       FileExt = ".py"
	ExtGO       FileExt = ".go"
	ExtRS       FileExt = ".rs"
	ExtJAVA     FileExt = ".java"
	ExtC        FileExt = ".c"
	ExtCPP      FileExt = ".cpp"
	ExtH        FileExt = ".h"
	ExtHPP      FileExt = ".hpp"
	ExtCS       FileExt = ".cs"
	ExtRB       FileExt = ".rb"
	ExtPHP      FileExt = ".php"
	ExtHTML     FileExt = ".html"
	ExtHTM      FileExt = ".htm"
	ExtCSS      FileExt = ".css"
	ExtSCSS     FileExt = ".scss"
	ExtLESS     FileExt = ".less"
	ExtSQL      FileExt = ".sql"
	ExtMod      FileExt = ".mod"
	ExtSum      FileExt = ".sum"
	ExtJSONL    FileExt = ".jsonl"
	ExtShell    FileExt = ".sh"
	ExtSWIFT    FileExt = ".swift"
	ExtM        FileExt = ".m"
	ExtKT       FileExt = ".kt"
	ExtPL       FileExt = ".pl"
	ExtSCALA    FileExt = ".scala"
	ExtHS       FileExt = ".hs"
	ExtLUA      FileExt = ".lua"
	ExtDART     FileExt = ".dart"
	ExtCmake    FileExt = ".cmake"
	ExtBazel    FileExt = ".bazel"

	ExtJPG  FileExt = ".jpg"
	ExtJPEG FileExt = ".jpeg"
	ExtPNG  FileExt = ".png"
	ExtGIF  FileExt = ".gif"
	ExtWEBP FileExt = ".webp"
	ExtBMP  FileExt = ".bmp"
	ExtSVG  FileExt = ".svg"

	ExtPDF  FileExt = ".pdf"
	ExtDOC  FileExt = ".doc"
	ExtDOCX FileExt = ".docx"
	ExtPPT  FileExt = ".ppt"
	ExtPPTX FileExt = ".pptx"
	ExtXLS  FileExt = ".xls"
	ExtXLSX FileExt = ".xlsx"
	ExtODT  FileExt = ".odt"
	ExtODS  FileExt = ".ods"
)

type MIMEType string

const (
	MIMEEmpty                  MIMEType = ""
	MIMEApplicationOctetStream MIMEType = "application/octet-stream"

	MIMETextPlain       MIMEType = "text/plain; charset=utf-8"
	MIMETextMarkdown    MIMEType = "text/markdown; charset=utf-8"
	MIMETextHTML        MIMEType = "text/html; charset=utf-8"
	MIMETextCSS         MIMEType = "text/css; charset=utf-8"
	MIMEApplicationJSON MIMEType = "application/json"
	MIMEApplicationYAML MIMEType = "application/x-yaml"
	MIMEApplicationTOML MIMEType = "application/toml"
	MIMEApplicationSQL  MIMEType = "application/sql"
	MIMEApplicationJS   MIMEType = "application/javascript"

	MIMEImageJPEG MIMEType = "image/jpeg"
	MIMEImagePNG  MIMEType = "image/png"
	MIMEImageGIF  MIMEType = "image/gif"
	MIMEImageWEBP MIMEType = "image/webp"
	MIMEImageBMP  MIMEType = "image/bmp"
	MIMEImageSVG  MIMEType = "image/svg+xml"

	MIMEApplicationPDF        MIMEType = "application/pdf"
	MIMEApplicationMSWord     MIMEType = "application/msword"
	MIMEApplicationMSPowerPt  MIMEType = "application/vnd.ms-powerpoint"
	MIMEApplicationMSExcel    MIMEType = "application/vnd.ms-excel"
	MIMEApplicationOpenXMLDoc MIMEType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MIMEApplicationOpenXMLPPT MIMEType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	MIMEApplicationOpenXMLXLS MIMEType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MIMEApplicationODT        MIMEType = "application/vnd.oasis.opendocument.text"
	MIMEApplicationODS        MIMEType = "application/vnd.oasis.opendocument.spreadsheet"
)

var MIMETypeToExtensionMode = map[MIMEType]ExtensionMode{
	MIMEEmpty:                  ExtensionModeDefault,
	MIMEApplicationOctetStream: ExtensionModeDefault,

	MIMETextPlain:       ExtensionModeText,
	MIMETextMarkdown:    ExtensionModeText,
	MIMETextHTML:        ExtensionModeText,
	MIMETextCSS:         ExtensionModeText,
	MIMEApplicationJSON: ExtensionModeText,
	MIMEApplicationYAML: ExtensionModeText,
	MIMEApplicationTOML: ExtensionModeText,
	MIMEApplicationSQL:  ExtensionModeText,
	MIMEApplicationJS:   ExtensionModeText,

	MIMEImageJPEG: ExtensionModeImage,
	MIMEImagePNG:  ExtensionModeImage,
	MIMEImageGIF:  ExtensionModeImage,
	MIMEImageWEBP: ExtensionModeImage,
	MIMEImageBMP:  ExtensionModeImage,
	MIMEImageSVG:  ExtensionModeImage,

	MIMEApplicationPDF:        ExtensionModeDocument,
	MIMEApplicationMSWord:     ExtensionModeDocument,
	MIMEApplicationMSPowerPt:  ExtensionModeDocument,
	MIMEApplicationMSExcel:    ExtensionModeDocument,
	MIMEApplicationOpenXMLDoc: ExtensionModeDocument,
	MIMEApplicationOpenXMLPPT: ExtensionModeDocument,
	MIMEApplicationOpenXMLXLS: ExtensionModeDocument,
	MIMEApplicationODT:        ExtensionModeDocument,
	MIMEApplicationODS:        ExtensionModeDocument,
}

var ExtensionToMIMEType = map[FileExt]MIMEType{
	ExtTxt:      MIMETextPlain,
	ExtMd:       MIMETextMarkdown,
	ExtMarkdown: MIMETextMarkdown,
	ExtMdx:      MIMETextMarkdown,
	ExtLog:      MIMETextPlain,
	ExtJSON:     MIMEApplicationJSON,
	ExtYAML:     MIMEApplicationYAML,
	ExtYML:      MIMEApplicationYAML,
	ExtTOML:     MIMEApplicationTOML,
	ExtJS:       MIMEApplicationJS,
	ExtTS:       MIMETextPlain,
	ExtTSX:      MIMETextPlain,
	ExtJSX:      MIMETextPlain,
	ExtPY:       MIMETextPlain,
	ExtGO:       MIMETextPlain,
	ExtRS:       MIMETextPlain,
	ExtJAVA:     MIMETextPlain,
	ExtC:        MIMETextPlain,
	ExtCPP:      MIMETextPlain,
	ExtH:        MIMETextPlain,
	ExtHPP:      MIMETextPlain,
	ExtCS:       MIMETextPlain,
	ExtRB:       MIMETextPlain,
	ExtPHP:      MIMETextPlain,
	ExtHTML:     MIMETextHTML,
	ExtHTM:      MIMETextHTML,
	ExtCSS:      MIMETextCSS,
	ExtSCSS:     MIMETextPlain,
	ExtLESS:     MIMETextPlain,
	ExtSQL:      MIMEApplicationSQL,
	ExtMod:      MIMETextPlain,
	ExtSum:      MIMETextPlain,
	ExtJSONL:    MIMETextPlain,
	ExtShell:    MIMETextPlain,
	ExtSWIFT:    MIMETextPlain,
	ExtM:        MIMETextPlain,
	ExtKT:       MIMETextPlain,
	ExtPL:       MIMETextPlain,
	ExtSCALA:    MIMETextPlain,
	ExtHS:       MIMETextPlain,
	ExtLUA:      MIMETextPlain,
	ExtDART:     MIMETextPlain,
	ExtCmake:    MIMETextPlain,
	ExtBazel:    MIMETextPlain,

	ExtJPG:  MIMEImageJPEG,
	ExtJPEG: MIMEImageJPEG,
	ExtPNG:  MIMEImagePNG,
	ExtGIF:  MIMEImageGIF,
	ExtWEBP: MIMEImageWEBP,
	ExtBMP:  MIMEImageBMP,
	ExtSVG:  MIMEImageSVG,

	ExtPDF:  MIMEApplicationPDF,
	ExtDOC:  MIMEApplicationMSWord,
	ExtDOCX: MIMEApplicationOpenXMLDoc,
	ExtPPT:  MIMEApplicationMSPowerPt,
	ExtPPTX: MIMEApplicationOpenXMLPPT,
	ExtXLS:  MIMEApplicationMSExcel,
	ExtXLSX: MIMEApplicationOpenXMLXLS,
	ExtODT:  MIMEApplicationODT,
	ExtODS:  MIMEApplicationODS,
}

var ModeToExtensions = func() map[ExtensionMode][]FileExt {
	m := make(map[ExtensionMode][]FileExt, len(AllExtensionModes))

	for ext, mimeType := range ExtensionToMIMEType {
		if mode, ok := MIMETypeToExtensionMode[mimeType]; ok {
			m[mode] = append(m[mode], ext)
		} else {
			m[ExtensionModeDefault] = append(m[ExtensionModeDefault], ext)
		}
	}
	return m
}()
