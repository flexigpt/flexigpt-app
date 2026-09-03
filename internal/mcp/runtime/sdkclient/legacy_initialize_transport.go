package sdkclient

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

const sdkMethodServerDiscover = "server/discover"

// preferLegacyInitializeClient is an app-side compatibility shim.
//
// The current upstream MCP Go SDK defaults to the sessionless protocol and
// probes "server/discover" before falling back to legacy "initialize". Some
// older MCP servers hang on unknown methods instead of returning method-not-
// found, which makes Client.Connect wait until the whole connect timeout.
//
// This implementation suppresses only the outgoing server/discover call in
// client middleware. That is important: do not wrap the transport connection
// for this purpose. The streamable HTTP transport's concrete connection type
// receives an SDK-internal sessionUpdated callback after initialize. Wrapping
// it hides that private interface from the SDK, preventing protocol-version
// headers and standalone SSE setup from being initialized.
func preferLegacyInitializeClient(client *mcpSDK.Client) {
	if client == nil {
		return
	}

	client.AddSendingMiddleware(func(next mcpSDK.MethodHandler) mcpSDK.MethodHandler {
		return func(ctx context.Context, method string, req mcpSDK.Request) (mcpSDK.Result, error) {
			if method == sdkMethodServerDiscover {
				return nil, &jsonrpc.Error{
					Code: jsonrpc.CodeMethodNotFound,
					Message: "server/discover suppressed by FlexiGPT legacy-initialize " +
						"compatibility middleware",
				}
			}
			return next(ctx, method, req)
		}
	})
}
