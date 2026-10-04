package overlay

var (
	_ API      = (*Service)(nil)
	_ StoreAPI = (*Service)(nil)
)
