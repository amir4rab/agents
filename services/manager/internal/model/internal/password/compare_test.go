package password_test

import (
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/internal/password"
)

func TestVerify(t *testing.T) {
	val := "somepassword"

	original, err := password.CreateHash(val)
	if err != nil {
		t.Error(err)
	}

	ok, err := password.ComparePasswordAndHash("someothervalue", *original)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("Expected the passwords not to match")
	}

	ok, err = password.ComparePasswordAndHash(val, *original)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("Expected the verification to pass")
	}
}
