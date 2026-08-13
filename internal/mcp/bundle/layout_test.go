package bundle

import "testing"

func TestMCPPackageLayoutUsesCallerOwnedConvention(t *testing.T) {
	t.Parallel()

	address, err := PackageAddressForBundle("base", "")
	if err != nil {
		t.Fatalf("PackageAddressForBundle: %v", err)
	}

	directory, err := address.Directory()
	if err != nil {
		t.Fatalf("Directory: %v", err)
	}
	if directory != "mcp.bundle/base/unversioned" {
		t.Fatalf("directory=%q", directory)
	}

	locator, err := DocumentLocatorForPackage(address)
	if err != nil {
		t.Fatalf("DocumentLocatorForPackage: %v", err)
	}
	if locator != "mcp.bundle/base/unversioned/mcps.json" {
		t.Fatalf("locator=%q", locator)
	}

	if !IsBundleDocumentLocator(locator) {
		t.Fatal("canonical MCP document locator was not recognized")
	}
}
