package sdkclient

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

const sdkMethodServerDiscover = "server/discover"

// preferLegacyInitializeTransport is an app-side compatibility shim.
//
// The current upstream MCP Go SDK defaults to the sessionless protocol and
// probes "server/discover" before falling back to legacy "initialize". Some
// older MCP servers hang on unknown methods instead of returning method-not-
// found, which makes Client.Connect wait until the whole connect timeout.
//
// The SDK does not currently expose its ClientSessionOptions.protocolVersion
// field, so the app cannot ask it to start directly with 2025-11-25. Instead,
// this wrapper locally answers only the SDK's "server/discover" probe with a
// JSON-RPC method-not-found error. The SDK then follows its built-in legacy
// initialize fallback immediately.
//
// Remove this wrapper when the SDK exposes a public protocol-version option or
// when FlexiGPT has a per-server compatibility setting for sessionless MCP.
func preferLegacyInitializeTransport(inner mcpSDK.Transport) mcpSDK.Transport {
	if inner == nil {
		return nil
	}
	return &legacyInitializeTransport{inner: inner}
}

type legacyInitializeTransport struct {
	inner mcpSDK.Transport
}

func (t *legacyInitializeTransport) Connect(ctx context.Context) (mcpSDK.Connection, error) {
	if t == nil || t.inner == nil {
		return nil, errors.New("legacy initialize transport: nil inner transport")
	}

	delegate, err := t.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}

	readCtx, cancel := context.WithCancel(context.Background())
	conn := &legacyInitializeConn{
		delegate: delegate,
		readCtx:  readCtx,
		cancel:   cancel,
		incoming: make(chan legacyInitializeReadResult, 32),
		done:     make(chan struct{}),
	}
	go conn.readLoop()
	return conn, nil
}

type legacyInitializeReadResult struct {
	msg jsonrpc.Message
	err error
}

type legacyInitializeConn struct {
	delegate mcpSDK.Connection

	readCtx context.Context
	cancel  context.CancelFunc

	incoming chan legacyInitializeReadResult
	done     chan struct{}

	closeOnce sync.Once
	closeErr  error
}

func (c *legacyInitializeConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, io.EOF
	case result := <-c.incoming:
		return result.msg, result.err
	}
}

func (c *legacyInitializeConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	if req, ok := msg.(*jsonrpc.Request); ok && req.IsCall() && req.Method == sdkMethodServerDiscover {
		resp := &jsonrpc.Response{
			ID: req.ID,
			Error: &jsonrpc.Error{
				Code:    jsonrpc.CodeMethodNotFound,
				Message: "server/discover suppressed by FlexiGPT legacy-initialize compatibility wrapper",
			},
		}
		select {
		case c.incoming <- legacyInitializeReadResult{msg: resp}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return io.EOF
		}
	}

	return c.delegate.Write(ctx, msg)
}

func (c *legacyInitializeConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.cancel != nil {
			c.cancel()
		}
		if c.delegate != nil {
			c.closeErr = c.delegate.Close()
		}
	})
	return c.closeErr
}

func (c *legacyInitializeConn) SessionID() string {
	if c == nil || c.delegate == nil {
		return ""
	}
	return c.delegate.SessionID()
}

func (c *legacyInitializeConn) readLoop() {
	for {
		msg, err := c.delegate.Read(c.readCtx)
		select {
		case c.incoming <- legacyInitializeReadResult{msg: msg, err: err}:
		case <-c.done:
			return
		}
		if err != nil {
			return
		}
	}
}
