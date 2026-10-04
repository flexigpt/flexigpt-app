package ingest

var (
	_ Scanner                   = (*Engine)(nil)
	_ CompiledDocumentRegistrar = (*Engine)(nil)
)
