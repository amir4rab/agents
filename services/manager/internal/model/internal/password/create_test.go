package password_test

import (
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/internal/password"
)

func TestEncode(t *testing.T) {
	val := "somepassword"

	encoded, err := password.CreateHash(val)
	if err != nil {
		t.Error(err)
	}

	if *encoded == "" {
		t.Error("Expected that the encoded value not to be empty")
	}
}
