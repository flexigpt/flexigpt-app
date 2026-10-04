package refresh

var (
	_ API                       = (*Service)(nil)
	_ MetadataInspector         = (*Service)(nil)
	_ CompiledDocumentRegistrar = (*Service)(nil)
)
