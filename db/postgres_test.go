package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttemptLock(t *testing.T) {
	err := Init("postgres://pglock:pglock@localhost:5432/pglock?sslmode=disable", 1)
	if err != nil {
		t.FailNow()
	}
	var locked bool
	Truncate()
	locked, err = AttemptLock("123", "123")
	assert.NoError(t, err)
	assert.True(t, locked)
	locked, err = AttemptLock("123", "123")
	assert.NoError(t, err)
	assert.True(t, locked)
	locked, err = AttemptLock("123", "234")
	assert.NoError(t, err)
	assert.False(t, locked)
}
