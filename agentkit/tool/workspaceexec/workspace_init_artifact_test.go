//
// Tencent is pleased to support the open source community by making
// trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package workspaceexec

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/LingByte/ling-base/agentkit/agent"
	"github.com/LingByte/ling-base/agentkit/artifact"
	"github.com/LingByte/ling-base/agentkit/artifact/inmemory"
	"github.com/LingByte/ling-base/agentkit/codeexecutor"
	localexec "github.com/LingByte/ling-base/agentkit/codeexecutor/local"
	"github.com/LingByte/ling-base/agentkit/session"
)

// TestExecTool_WorkspaceInitHook_ArtifactInput end-to-end: resolver injects
// artifact context; init hook StageInputs(artifact://) runs during
// CreateWorkspace; user command sees the staged file.
func TestExecTool_WorkspaceInitHook_ArtifactInput(t *testing.T) {
	root := t.TempDir()
	svc := inmemory.NewService()
	sess := &session.Session{
		ID:      "sess-ws-init-art",
		AppName: "myapp",
		UserID:  "user1",
	}
	info := artifact.SessionInfo{
		AppName:   sess.AppName,
		UserID:    sess.UserID,
		SessionID: sess.ID,
	}
	_, err := svc.SaveArtifact(
		context.Background(),
		info,
		"app/requirements.txt",
		&artifact.Artifact{Data: []byte("numpy==1.26.4\n")},
	)
	require.NoError(t, err)

	exec, err2 := codeexecutor.NewWorkspaceInitExecutor(
		localexec.New(localexec.WithWorkDir(root)),
		codeexecutor.NewWorkspaceInitHook(codeexecutor.WorkspaceInitSpec{
			Inputs: []codeexecutor.InputSpec{{
				From: "artifact://app/requirements.txt@0",
				To:   "work/requirements.txt",
				Mode: "copy",
			}},
		}),
	)
	require.NoError(t, err2)

	tl := NewExecTool(exec)

	inv := agent.NewInvocation()
	inv.Session = sess
	inv.ArtifactService = svc
	ctx := agent.NewInvocationContext(context.Background(), inv)

	args := execInput{
		Command: "cat work/requirements.txt",
		Timeout: 60,
	}
	enc, err := json.Marshal(args)
	require.NoError(t, err)

	out, err := tl.Call(ctx, enc)
	require.NoError(t, err)
	eo, ok := out.(execOutput)
	require.True(t, ok)
	require.Equal(t, codeexecutor.ProgramStatusExited, eo.Status)
	require.Contains(t, eo.Output, "numpy==1.26.4")
}
