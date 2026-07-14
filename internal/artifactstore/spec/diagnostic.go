package spec

// DiagnosticLocation identifies an optional source position for a diagnostic.
// It must never contain an absolute filesystem path.
type DiagnosticLocation struct {
	Locator            SourceLocator      `json:"locator,omitempty"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`
	Line               int                `json:"line,omitempty"`
	Column             int                `json:"column,omitempty"`
}

// Diagnostic is a current, structured observation attached to a source,
// package, catalog resource, record, or catalog generation.
type Diagnostic struct {
	Severity DiagnosticSeverity  `json:"severity"`
	Code     string              `json:"code"`
	Message  string              `json:"message"`
	Location *DiagnosticLocation `json:"location,omitempty"`
}
