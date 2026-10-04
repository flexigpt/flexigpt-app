package resource

var (
	_ API           = (*ordinaryService)(nil)
	_ NativePathAPI = (*nativePathService)(nil)
)
