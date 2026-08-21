// Package extproto defines the JSON-over-stdin/stdout wire format
// spoken between LingAgent and its extension subprocesses. Both the
// host (packages/extensions) and the SDK (packages/ext) marshal/
// unmarshal the same types, so changes here ripple through both.
//
// All frames are one JSON object terminated by a single LF. Object
// boundaries follow newline boundaries; no multi-line JSON.
//
// Direction conventions:
//   - Type names ending in "FromExt" are sent by the extension to LingAgent.
//   - Type names ending in "FromHost" are sent by LingAgent to the extension.
//
// Every frame has a top-level Type discriminator. Optional ID is
// present on commands and responses so the sender can correlate.
package extproto

import "encoding/json"

// ProtocolVersion is the wire protocol version. Bumped on breaking changes.
const ProtocolVersion = 1

// Frame is the base type all frames embed.
type Frame struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

// ---- Extension → Host ----

// HelloFromExt is the first frame an extension sends on startup.
type HelloFromExt struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// RegisterCommandFromExt registers a slash command.
type RegisterCommandFromExt struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// RegisterToolFromExt registers a tool the model can call.
type RegisterToolFromExt struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Deferred    bool            `json:"deferred,omitempty"`
}

// ReadyFromExt signals the extension has finished registering.
type ReadyFromExt struct {
	Type string `json:"type"`
}

// SubscribeFromExt subscribes to host events.
type SubscribeFromExt struct {
	Type   string   `json:"type"`
	Events []string `json:"events,omitempty"`
}

// ToolResultFromExt is the result of a tool call.
type ToolResultFromExt struct {
	Type    string         `json:"type"`
	ID      string         `json:"id"`
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"is_error,omitempty"`
}

// ContentBlock is one block of tool result content.
type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     string `json:"data,omitempty"`
}

// CommandResponseFromExt is the response to a slash command invocation.
type CommandResponseFromExt struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Action  string `json:"action"` // "prompt" | "insert" | "display" | "error"
	Prompt  string `json:"prompt,omitempty"`
	Insert  string `json:"insert,omitempty"`
	Display string `json:"display,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NotifyFromExt sends a notification to the host UI.
type NotifyFromExt struct {
	Type    string `json:"type"`
	Level   string `json:"level"` // "info" | "warn" | "error"
	Message string `json:"message"`
}

// ShutdownAckFromExt acknowledges a shutdown request.
type ShutdownAckFromExt struct {
	Type string `json:"type"`
}

// ---- Host → Extension ----

// HelloAckFromHost is the host's response to HelloFromExt.
type HelloAckFromHost struct {
	Type            string `json:"type"`
	ProtocolVersion int    `json:"protocol_version"`
	HostVersion     string `json:"host_version"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	CWD             string `json:"cwd"`
	ExtensionDir    string `json:"extension_dir,omitempty"`
	DataDir         string `json:"data_dir,omitempty"`
}

// CommandInvokedFromHost tells the extension a slash command was used.
type CommandInvokedFromHost struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args,omitempty"`
}

// ToolCallFromHost tells the extension to execute a tool.
type ToolCallFromHost struct {
	Type string          `json:"type"`
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// EventFromHost notifies the extension of a lifecycle event.
type EventFromHost struct {
	Type     string `json:"type"`
	Event    string `json:"event"`
	Step     int    `json:"step,omitempty"`
	Text     string `json:"text,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
	Error    string `json:"error,omitempty"`
}

// ShutdownFromHost asks the extension to shut down cleanly.
type ShutdownFromHost struct {
	Type string `json:"type"`
}

// Encode marshals v as JSON and appends a newline.
func Encode(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
