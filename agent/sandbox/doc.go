// Package sandbox exposes ShellRunner for ling-agent tools.
//
// OS sandboxing (seatbelt / bwrap / seccomp) is provided by
// agentkit/codeexecutor/sandbox; this package keeps the CLI-facing
// ShellRunner and docker/podman Container adapters.
package sandbox
