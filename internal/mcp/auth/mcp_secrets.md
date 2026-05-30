# MCP secrets

MCP server configs must not contain raw secrets. They store deterministic secret
refs, and the secret values are stored in the settings auth-key store under the
`mcp` auth-key namespace.

OAuth access tokens and refresh tokens are not persisted. They live only in
SDK token sources in the current process. Restarting the app forces OAuth
authorization again.

## Storage model

A secret ref is a string with this shape:

    mcpv1:<base64url-canonical-json>

The canonical JSON contains:

    {
      "serverID": "server-id",
      "kind": "stdioEnv",
      "slot": "github_token"
    }

or:

    {
      "serverID": "server-id",
      "kind": "oauthClientCredentials",
      "slot": "clientcredentials"
    }

The actual stored setting key is not the ref itself. It is:

    mcpv1:<sha256(canonical-secret-ref-json)>

The setting store path is:

    authKeys / mcp / <storage-key> / secret

The setting store encrypts the `secret` value on disk.

## Backend helper endpoints

The frontend should call backend helper methods to create/delete MCP secrets. It
must not construct `mcpv1:` refs manually.

### Create or update a secret

Request body:

    {
      "kind": "stdioEnv",
      "slot": "GITHUB_TOKEN",
      "secret": "ghp_example"
    }

Response body:

    {
      "secretRef": "mcpv1:...",
      "sha256": "...",
      "nonEmpty": true
    }

### Delete a secret

Delete by deterministic `serverID`, `kind`, and `slot`.

## Stdio env secrets

Use `kind: "stdioEnv"` and `slot` equal to the environment variable name.

Create secret request:

    {
      "kind": "stdioEnv",
      "slot": "GITHUB_TOKEN",
      "secret": "ghp_example"
    }

Server config:

    {
      "transport": "stdio",
      "stdio": {
        "command": "my-mcp-server",
        "secretEnvRefs": {
          "GITHUB_TOKEN": "mcpv1:..."
        }
      }
    }

At runtime, the backend resolves the secret ref and injects it into the child
process environment. The process does not inherit the full host environment.

## OAuth authorization-code, public client

Public OAuth clients use PKCE and do not require a client secret.

Secret value:

    {
      "clientID": "public-client-id"
    }

Server config:

    {
      "transport": "streamableHttp",
      "streamableHttp": {
        "url": "https://example.com/mcp",
        "authMode": "oauth",
        "clientCredentialRef": "mcpv1:..."
      }
    }

## OAuth authorization-code, confidential client

Confidential OAuth clients include a client secret.

Secret value:

    {
      "clientID": "confidential-client-id",
      "clientSecret": "client-secret"
    }

Server config:

    {
      "transport": "streamableHttp",
      "streamableHttp": {
        "url": "https://example.com/mcp",
        "authMode": "oauth",
        "clientCredentialRef": "mcpv1:..."
      }
    }

## OAuth dynamic client registration

If no `clientCredentialRef` is configured for `authMode: "oauth"`, the backend
uses the official MCP Go SDK dynamic client registration flow when the server's
authorization server supports it.

Server config:

    {
      "transport": "streamableHttp",
      "streamableHttp": {
        "url": "https://example.com/mcp",
        "authMode": "oauth"
      }
    }

Dynamically issued client credentials are not persisted by FlexiGPT.

## OAuth Client ID Metadata Document

The official MCP Go SDK supports Client ID Metadata Document registration.

Server config:

    {
      "transport": "streamableHttp",
      "streamableHttp": {
        "url": "https://example.com/mcp",
        "authMode": "oauth",
        "clientIDMetadataDocumentURL": "https://client.example.com/flexigpt-mcp-client.json"
      }
    }

If the authorization server does not support Client ID Metadata Documents and
dynamic client registration is available, the SDK can fall back to DCR.

## OAuth client credentials grant

The client credentials grant requires a confidential client.

Secret value:

    {
      "clientID": "service-client-id",
      "clientSecret": "service-client-secret"
    }

Server config:

    {
      "transport": "streamableHttp",
      "streamableHttp": {
        "url": "https://example.com/mcp",
        "authMode": "clientCredentials",
        "clientCredentialRef": "mcpv1:..."
      }
    }

The SDK obtains access tokens using the standard client credentials grant and
refreshes/replaces tokens in memory when they expire.

## Prohibited patterns

Do not:

- Put secrets in `streamableHttp.url`.
- Use URL userinfo in MCP HTTP URLs.
- Store OAuth access tokens in server config.
- Store OAuth refresh tokens in server config.
- Hand-build `mcpv1:` refs in frontend code.
- Log raw secret values.

## Redaction

The backend redacts configured sensitive values from:

- MCP stdio server stderr lines.
- OAuth auth-status errors emitted by token or authorization failures.

OAuth tokens are not intentionally logged or persisted.
