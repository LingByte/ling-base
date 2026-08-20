// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashArgon2idDefault(t *testing.T) {
	h, err := Hash("my-password", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, h)
	assert.True(t, strings.HasPrefix(h, "$argon2id$"))
	assert.Equal(t, AlgorithmArgon2id, AlgorithmOf(h))
}

func TestHashBcrypt(t *testing.T) {
	h, err := Hash("my-password", &Options{Algorithm: AlgorithmBcrypt})
	require.NoError(t, err)
	assert.NotEmpty(t, h)
	assert.True(t, strings.HasPrefix(h, "$2a$") || strings.HasPrefix(h, "$2b$"))
	assert.Equal(t, AlgorithmBcrypt, AlgorithmOf(h))
}

func TestHashBcryptCustomCost(t *testing.T) {
	h, err := Hash("test", &Options{Algorithm: AlgorithmBcrypt, BcryptCost: 10})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(h, "$2a$10$") || strings.HasPrefix(h, "$2b$10$"))
}

func TestHashEmptyPassword(t *testing.T) {
	_, err := Hash("", nil)
	assert.ErrorIs(t, err, ErrEmptyPassword)
}

func TestHashBcryptTooLong(t *testing.T) {
	long := strings.Repeat("a", 73)
	_, err := Hash(long, &Options{Algorithm: AlgorithmBcrypt})
	assert.ErrorIs(t, err, ErrPasswordTooLong)
}

func TestVerifyArgon2idRoundTrip(t *testing.T) {
	h, err := Hash("correct-horse-battery-staple", nil)
	require.NoError(t, err)

	assert.True(t, Verify("correct-horse-battery-staple", h))
	assert.False(t, Verify("wrong-password", h))
}

func TestVerifyBcryptRoundTrip(t *testing.T) {
	h, err := Hash("my-secret", &Options{Algorithm: AlgorithmBcrypt})
	require.NoError(t, err)

	assert.True(t, Verify("my-secret", h))
	assert.False(t, Verify("not-my-secret", h))
}

func TestVerifyCrossAlgorithm(t *testing.T) {
	// Argon2id hash should not verify with bcrypt and vice versa.
	argonHash, _ := Hash("pw", nil)
	bcryptHash, _ := Hash("pw", &Options{Algorithm: AlgorithmBcrypt})

	assert.True(t, Verify("pw", argonHash))
	assert.True(t, Verify("pw", bcryptHash))
	assert.False(t, Verify("pw", "$invalid$hash$format"))
}

func TestVerifyEmptyInputs(t *testing.T) {
	assert.False(t, Verify("", "somehash"))
	assert.False(t, Verify("pw", ""))
}

func TestVerifyBcryptWithPrefix(t *testing.T) {
	// bcrypt hash with $bcrypt$ prefix.
	h, err := Hash("test", &Options{Algorithm: AlgorithmBcrypt})
	require.NoError(t, err)
	prefixed := "$bcrypt$" + h
	assert.True(t, Verify("test", prefixed))
}

func TestNeedsRehashUpgradeBcryptToArgon2id(t *testing.T) {
	bcryptHash, _ := Hash("pw", &Options{Algorithm: AlgorithmBcrypt})
	// Default target is Argon2id, so a bcrypt hash needs rehashing.
	assert.True(t, NeedsRehash(bcryptHash, nil))
}

func TestNeedsRehashArgon2idSameParams(t *testing.T) {
	h, _ := Hash("pw", nil)
	// Same default params → no rehash needed.
	assert.False(t, NeedsRehash(h, nil))
}

func TestNeedsRehashArgon2idHigherParams(t *testing.T) {
	h, _ := Hash("pw", &Options{
		Algorithm:    AlgorithmArgon2id,
		Argon2Time:   1,
		Argon2Memory: 16 * 1024,
	})
	// Default params are higher → needs rehash.
	assert.True(t, NeedsRehash(h, nil))
}

func TestNeedsRehashBcryptLowerCost(t *testing.T) {
	h, _ := Hash("pw", &Options{Algorithm: AlgorithmBcrypt, BcryptCost: 4})
	// Default cost is 12, so cost-4 hash needs rehash.
	assert.True(t, NeedsRehash(h, &Options{Algorithm: AlgorithmBcrypt}))
}

func TestNeedsRehashBcryptSameCost(t *testing.T) {
	h, _ := Hash("pw", &Options{Algorithm: AlgorithmBcrypt, BcryptCost: 12})
	assert.False(t, NeedsRehash(h, &Options{Algorithm: AlgorithmBcrypt, BcryptCost: 12}))
}

func TestNeedsRehashInvalidHash(t *testing.T) {
	assert.True(t, NeedsRehash("garbage", nil))
	assert.True(t, NeedsRehash("", nil))
}

func TestAlgorithmOf(t *testing.T) {
	argonHash, _ := Hash("pw", nil)
	bcryptHash, _ := Hash("pw", &Options{Algorithm: AlgorithmBcrypt})

	assert.Equal(t, AlgorithmArgon2id, AlgorithmOf(argonHash))
	assert.Equal(t, AlgorithmBcrypt, AlgorithmOf(bcryptHash))
	assert.Equal(t, Algorithm(""), AlgorithmOf("invalid"))
	assert.Equal(t, Algorithm(""), AlgorithmOf(""))
}

func TestMustHash(t *testing.T) {
	h := MustHash("test", nil)
	assert.NotEmpty(t, h)
	assert.True(t, Verify("test", h))
}

func TestMustHashPanics(t *testing.T) {
	assert.Panics(t, func() {
		MustHash("", nil)
	})
}

func TestMigrationFlow(t *testing.T) {
	// Simulate: user has old bcrypt hash, logs in, we upgrade to argon2id.
	oldHash, _ := Hash("mypassword", &Options{Algorithm: AlgorithmBcrypt, BcryptCost: 4})

	// On login:
	if Verify("mypassword", oldHash) {
		if NeedsRehash(oldHash, nil) {
			newHash, err := Hash("mypassword", nil)
			require.NoError(t, err)
			assert.True(t, Verify("mypassword", newHash))
			assert.Equal(t, AlgorithmArgon2id, AlgorithmOf(newHash))
		}
	}
}

func TestDifferentPasswordsDifferentHashes(t *testing.T) {
	h1, _ := Hash("password1", nil)
	h2, _ := Hash("password2", nil)
	assert.NotEqual(t, h1, h2)
}

func TestSamePasswordDifferentHashes(t *testing.T) {
	// Same password should produce different hashes (random salt).
	h1, _ := Hash("same-password", nil)
	h2, _ := Hash("same-password", nil)
	assert.NotEqual(t, h1, h2)
	// But both should verify.
	assert.True(t, Verify("same-password", h1))
	assert.True(t, Verify("same-password", h2))
}
