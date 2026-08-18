package mcpbundle

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
)

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
	if string(directory) != string(
		artifactbuiltin.MCPBundlePackageKind,
	)+"/base/"+string(artifactbuiltin.UnversionedPackageVersion) {
		t.Fatalf("directory=%q", directory)
	}

	locator, err := DocumentLocatorForPackage(address)
	if err != nil {
		t.Fatalf("DocumentLocatorForPackage: %v", err)
	}
	if string(locator) != string(directory)+"/"+string(artifactbuiltin.MCPBundleDocumentFileName) {
		t.Fatalf("locator=%q", locator)
	}

	if !IsBundleDocumentLocator(locator) {
		t.Fatal("canonical MCP document locator was not recognized")
	}
}
