package sandbox

import (
	aksandbox "github.com/LingByte/ling-base/agentkit/codeexecutor/sandbox"
)

// Re-export the agentkit OS-sandbox core so ShellRunner can stay in this
// package while the heavy runtime lives in agentkit.

type (
	Runtime           = aksandbox.Runtime
	Option            = aksandbox.Option
	SessionPolicy     = aksandbox.SessionPolicy
	NetworkPolicy     = aksandbox.NetworkPolicy
	PermissionProfile = aksandbox.PermissionProfile
	ProcessSpec       = aksandbox.ProcessSpec
)

const (
	BackendAuto                  = aksandbox.BackendAuto
	NetworkEnabled               = aksandbox.NetworkEnabled
	NetworkRestricted            = aksandbox.NetworkRestricted
	SessionPersistencePerSession = aksandbox.SessionPersistencePerSession
)

var (
	NewRuntime              = aksandbox.NewRuntime
	WithBackend             = aksandbox.WithBackend
	WithPermissionProfile   = aksandbox.WithPermissionProfile
	WithSessionPolicy       = aksandbox.WithSessionPolicy
	DangerFullAccessProfile = aksandbox.DangerFullAccessProfile
	WorkspaceWriteProfile   = aksandbox.WorkspaceWriteProfile
)
