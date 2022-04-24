package jsonrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordStripped(t *testing.T) {
	url := "https://user:password@my.api.com/endpoint"
	stripped := stripPassword(url)
	assert.Equal(t, "https://user:***@my.api.com/endpoint", stripped)
}
