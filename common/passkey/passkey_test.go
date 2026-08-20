// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package passkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidConfig(t *testing.T) {
	pk, err := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	require.NoError(t, err)
	require.NotNil(t, pk)
}

func TestNewMissingRPID(t *testing.T) {
	_, err := New(Config{
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewMissingDisplayName(t *testing.T) {
	_, err := New(Config{
		RPID:      "example.com",
		RPOrigins: []string{"https://example.com"},
	})
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewMissingOrigins(t *testing.T) {
	_, err := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
	})
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNewEmptyOrigins(t *testing.T) {
	_, err := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{},
	})
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

// mockUser implements User for testing.
type mockUser struct {
	id          []byte
	name        string
	displayName string
	creds       []Credential
}

func (u *mockUser) WebAuthnID() []byte             { return u.id }
func (u *mockUser) WebAuthnName() string           { return u.name }
func (u *mockUser) WebAuthnDisplayName() string    { return u.displayName }
func (u *mockUser) WebAuthnIcon() string           { return "" }
func (u *mockUser) WebAuthnCredentials() []Credential { return u.creds }

func TestBeginRegistrationNilUser(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	_, _, err := pk.BeginRegistration(nil, nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestBeginRegistrationSuccess(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	user := &mockUser{
		id:          []byte("user-123"),
		name:        "alice@example.com",
		displayName: "Alice",
	}
	cred, session, err := pk.BeginRegistration(user, nil)
	require.NoError(t, err)
	require.NotNil(t, cred)
	require.NotNil(t, session)
	assert.NotEmpty(t, session.Challenge)
}

func TestBeginRegistrationWithExclusions(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	user := &mockUser{
		id:          []byte("user-123"),
		name:        "alice@example.com",
		displayName: "Alice",
	}
	cred, _, err := pk.BeginRegistration(user, &RegistrationOptions{
		ExcludeCredentials:  [][]byte{[]byte("existing-cred-id")},
		RequireResidentKey:  true,
	})
	require.NoError(t, err)
	require.NotNil(t, cred)
}

func TestBeginLoginNoCredentials(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	user := &mockUser{
		id:          []byte("user-123"),
		name:        "alice@example.com",
		displayName: "Alice",
		creds:       nil, // no credentials
	}
	_, _, err := pk.BeginLogin(user, nil)
	assert.ErrorIs(t, err, ErrNoCredentials)
}

func TestBeginLoginNilUser(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	_, _, err := pk.BeginLogin(nil, nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestBeginLoginWithCredentials(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	user := &mockUser{
		id:          []byte("user-123"),
		name:        "alice@example.com",
		displayName: "Alice",
		creds: []Credential{
			{ID: []byte("cred-1"), PublicKey: []byte("pk-1")},
		},
	}
	assertion, session, err := pk.BeginLogin(user, nil)
	require.NoError(t, err)
	require.NotNil(t, assertion)
	require.NotNil(t, session)
}

func TestBeginDiscoverableLogin(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	assertion, session, err := pk.BeginDiscoverableLogin(nil)
	require.NoError(t, err)
	require.NotNil(t, assertion)
	require.NotNil(t, session)
}

func TestFinishRegistrationNilSession(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	user := &mockUser{id: []byte("u1"), name: "a@b.com", displayName: "A"}
	_, err := pk.FinishRegistration(user, nil, nil)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestFinishRegistrationNilUser(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	_, err := pk.FinishRegistration(nil, &SessionData{}, nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestFinishDiscoverableLoginNilHandler(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	_, _, err := pk.FinishDiscoverableLogin(nil, &SessionData{}, nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestFinishDiscoverableLoginNilSession(t *testing.T) {
	pk, _ := New(Config{
		RPID:          "example.com",
		RPDisplayName: "My App",
		RPOrigins:     []string{"https://example.com"},
	})
	handler := func(rawID, userHandle []byte) (User, error) {
		return nil, ErrUserNotFound
	}
	_, _, err := pk.FinishDiscoverableLogin(handler, nil, nil)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestUserAdapter(t *testing.T) {
	u := &mockUser{
		id:          []byte("id-bytes"),
		name:        "name",
		displayName: "display",
		creds:       []Credential{{ID: []byte("c1")}},
	}
	adapted := adaptUser(u)
	assert.Equal(t, []byte("id-bytes"), adapted.WebAuthnID())
	assert.Equal(t, "name", adapted.WebAuthnName())
	assert.Equal(t, "display", adapted.WebAuthnDisplayName())
	assert.Len(t, adapted.WebAuthnCredentials(), 1)
}
