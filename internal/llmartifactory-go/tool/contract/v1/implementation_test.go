package toolv1

import "testing"

func TestToolDocumentSupportsGoAndSDKImplementations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  []byte
		want ImplementationKind
	}{
		{
			name: "go tool",
			raw: []byte(`{
				"type":"tool",
				"name":"go-tool",
				"version":"v1",
				"autoExecute":false,
				"inputSchema":{"type":"object"},
				"implementation":{
					"kind":"go",
					"function":"example.com/tools.GoTool"
				}
			}`),
			want: ImplementationKindGo,
		},
		{
			name: "sdk function tool",
			raw: []byte(`{
				"type":"tool",
				"name":"sdk-function-tool",
				"version":"v1",
				"autoExecute":false,
				"inputSchema":{"type":"object"},
				"userArgSchema":{"type":"object"},
				"implementation":{
					"kind":"sdk",
					"sdkType":"providerSDKTypeExample",
					"sdkToolType":"function"
				}
			}`),
			want: ImplementationKindSDK,
		},
		{
			name: "sdk custom tool",
			raw: []byte(`{
				"type":"tool",
				"name":"sdk-custom-tool",
				"version":"v1",
				"autoExecute":false,
				"inputSchema":{"type":"object"},
				"implementation":{
					"kind":"sdk",
					"sdkType":"providerSDKTypeExample",
					"sdkToolType":"custom"
				}
			}`),
			want: ImplementationKindSDK,
		},
		{
			name: "sdk web search tool",
			raw: []byte(`{
				"type":"tool",
				"name":"sdk-web-search-tool",
				"version":"v1",
				"autoExecute":false,
				"inputSchema":{"type":"object"},
				"implementation":{
					"kind":"sdk",
					"sdkType":"providerSDKTypeExample",
					"sdkToolType":"webSearch"
				}
			}`),
			want: ImplementationKindSDK,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			document, err := DecodeToolJSON(testCase.raw)
			if err != nil {
				t.Fatalf("DecodeToolJSON() error = %v", err)
			}
			if document.Implementation.Kind != testCase.want {
				t.Fatalf(
					"implementation kind = %q, want %q",
					document.Implementation.Kind,
					testCase.want,
				)
			}
		})
	}
}
