package store

import "testing"

func TestToolStoreListsOnlyBuiltInTools(t *testing.T) {
	store, err := NewToolStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewToolStore() error = %v", err)
	}
	t.Cleanup(store.Close)

	bundles, err := store.ListToolBundles(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatalf("ListToolBundles() error = %v", err)
	}
	for _, bundle := range bundles.Body.ToolBundles {
		if !bundle.IsBuiltIn {
			t.Fatalf("non-built-in bundle returned: %q", bundle.ID)
		}
	}

	tools, err := store.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Body.ToolListItems) == 0 {
		t.Fatal("ListTools() returned no built-in tools")
	}
	for _, tool := range tools.Body.ToolListItems {
		if !tool.IsBuiltIn || !tool.ToolDefinition.IsBuiltIn {
			t.Fatalf(
				"non-built-in tool returned: %s/%s@%s",
				tool.BundleID,
				tool.ToolSlug,
				tool.ToolVersion,
			)
		}
	}
}
