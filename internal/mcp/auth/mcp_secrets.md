# Artifact-backed MCP secrets

Artifact-backed MCP Server Definitions, MCP Bundle documents, Artifact local
data, Catalogs, and runtime snapshots must never contain raw secret values.
They contain only opaque Artifact-scoped secret references.

Secret values are stored through the existing Setting Store under the `mcp`
auth-key namespace. OAuth access and refresh tokens are application-local
secret values. They are never included in source documents, Definitions,
Artifact data, Catalogs, conversation records, or general runtime projections.

## Storage model

A secret ref is a string with this shape:

    mcpv2:<base64url-canonical-json>

The canonical JSON contains:

    {
      "server": {
        "rootID": "0198f097-0d5b-7000-8000-000000000002",
        "artifactID": "0198f097-0d5b-7000-8000-000000000010"
      },
      "kind": "httpHeader",
      "slot": "authorization"
    }

The actual Setting Store key is not the ref itself. It is:

    mcpv2:<sha256(canonical-secret-ref-json)>

The Setting Store namespace is:

    authKeys / mcp / <storage-key> / secret

The Setting Store encrypts the `secret` value on disk. `SetAuthKey` returns an
empty response body, so the MCP secret adapter reads the persisted metadata
after writing and returns the Setting Store's exact `SHA256` and `NonEmpty`
values. Callers must not recreate those values with different whitespace rules.

## Backend helper endpoints

The frontend must call Artifact-backed MCP wrapper methods to create or delete
MCP secrets. It must not construct `mcpv2:` references manually.

### Create or update a secret

Request body:

    {
      "kind": "stdioEnv",
      "slot": "GITHUB_TOKEN",
      "secret": "ghp_example"
    }

The wrapper receives the target `ArtifactRef` separately and returns:

    {
      "secretRef": "mcpv2:...",
      "sha256": "...",
      "nonEmpty": true
    }

### Delete a secret

Delete by target Server `ArtifactRef`, secret kind, and slot.

## Stdio env secrets

Use `kind: "stdioEnv"` and `slot` equal to the environment variable name.
The resolved value is substituted only into the materialized stdio environment.
It is not persisted into the server Definition.

The process receives only the materialized environment explicitly declared by
the resolved MCP server configuration. It does not inherit the full host
environment.

## OAuth authorization-code, public client

Public OAuth clients use PKCE and do not require a client secret.

Secret value:

    {
      "clientID": "public-client-id"
    }

The opaque reference is stored in Artifact local installation data only when
the server Definition declares an `oauthClientCredentials` installation input.

OAuth authorization-code flows may also use Dynamic Client Registration when
the server Definition permits it and no client credential input is required.

## OAuth authorization-code, confidential client

Confidential OAuth clients include a client secret.

Secret value:

    {
      "clientID": "confidential-client-id",
      "clientSecret": "client-secret"
    }

## OAuth dynamic client registration

If a canonical Server Definition has OAuth mode but no required client
credential input, the backend may use the MCP Go SDK dynamic client
registration flow when the authorization server supports it.

Dynamically issued client credentials are not written into portable MCP
documents or Artifact local data.

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

The client-credentials grant requires a confidential
`oauthClientCredentials` secret. The MCP wrapper validates that `clientSecret`
is present before accepting the secret for a server using
`auth.mode = "clientCredentials"`.

Secret value:

    {
      "clientID": "service-client-id",
      "clientSecret": "service-client-secret"
    }

The SDK obtains access tokens using the standard client-credentials grant.

## Prohibited patterns

Do not:

- Put secrets in a portable MCP URL.
- Use URL userinfo in MCP HTTP URLs.
- Store OAuth access tokens or refresh tokens in portable MCP configuration.
- Hand-build `mcpv2:` refs in frontend code.
- Log raw secret values.

## Redaction

The backend redacts configured sensitive values from:

- MCP stdio server stderr lines.
- OAuth auth-status and runtime errors emitted by token or authorization
  failures.

OAuth tokens are never intentionally logged.
