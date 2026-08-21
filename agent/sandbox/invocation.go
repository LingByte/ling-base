package sandbox

import "context"

// invocationSession is a minimal session representation used by the sandbox
// to derive a deterministic execution workspace id from the active invocation.
// It mirrors the subset of the framework's session.Session that
// executionIDFromContext inspects (AppName, UserID, ID).
type invocationSession struct {
	AppName string
	UserID  string
	ID      string
}

// invocation is a minimal invocation representation carrying the session
// needed for workspace id derivation. It mirrors the subset of
// the framework's agent.Invocation that executionIDFromContext inspects.
type invocation struct {
	Session *invocationSession
}

type invocationKey struct{}

// withInvocation attaches an invocation to ctx so executionIDFromContext can
// derive a workspace id. This is the sandbox-local equivalent of
// agent.NewInvocationContext.
func withInvocation(ctx context.Context, inv *invocation) context.Context {
	return context.WithValue(ctx, invocationKey{}, inv)
}

// invocationFromContext retrieves the invocation previously attached by
// withInvocation. This is the sandbox-local equivalent of
// agent.InvocationFromContext.
func invocationFromContext(ctx context.Context) (*invocation, bool) {
	inv, ok := ctx.Value(invocationKey{}).(*invocation)
	return inv, ok
}
