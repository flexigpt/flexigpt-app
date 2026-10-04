package source

var (
	_ API              = (*Service)(nil)
	_ ContentMutation  = (*Service)(nil)
	_ Runtime          = (*runtime)(nil)
	_ LocalPathRuntime = (*runtime)(nil)
)
