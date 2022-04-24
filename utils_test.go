package jsonrpc

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestPasswordStripped(t *testing.T) {
	url := "https://user:password@my.api.com/endpoint"
	stripped := stripPassword(url)
	Expect(stripped).To(Equal("https://user:***@my.api.com/endpoint"))
}
