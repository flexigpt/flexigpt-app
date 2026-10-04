package fsdir

const gitMetadataDirectoryName = ".git"

var defaultTraversalExcludedDirectoryNames = []string{
	".git",
	".hg",
	".svn",
	"node_modules",
	"vendor",
	"bower_components",
}
