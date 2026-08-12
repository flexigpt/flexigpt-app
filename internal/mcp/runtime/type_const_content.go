package runtime

import (
	"time"
)

const (
	NotificationRefreshDebounce = 1 * time.Second
	MaxMCPServerPageSize        = 256
	DefaultMCPPageSize          = 25
)

type MCPContentType string

const (
	MCPContentTypeText         MCPContentType = "text"
	MCPContentTypeImage        MCPContentType = "image"
	MCPContentTypeAudio        MCPContentType = "audio"
	MCPContentTypeResourceLink MCPContentType = "resource_link"
	MCPContentTypeResource     MCPContentType = "resource"
)

type MCPResourceContents struct {
	URI      string         `json:"uri"`
	MIMEType string         `json:"mimeType,omitempty"`
	Text     string         `json:"text,omitempty"`
	Blob     []byte         `json:"blob,omitempty"`
	Meta     map[string]any `json:"_meta,omitempty"`
}

type MCPIcon struct {
	Source   string   `json:"src"`
	MIMEType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
	Theme    string   `json:"theme,omitempty"`
}

type MCPContent struct {
	Type MCPContentType `json:"type"`

	Text     string `json:"text,omitempty"`
	Data     []byte `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`

	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Size        *int64 `json:"size,omitempty"`

	Resource *MCPResourceContents `json:"resource,omitempty"`

	Annotations map[string]any `json:"annotations,omitempty"`
	Meta        map[string]any `json:"_meta,omitempty"`
	Icons       []MCPIcon      `json:"icons,omitempty"`
}

type MCPPromptMessage struct {
	Role    string     `json:"role"`
	Content MCPContent `json:"content"`
}
