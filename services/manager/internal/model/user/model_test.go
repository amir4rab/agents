package user_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/user"
)

func TestUserRoleJSON(t *testing.T) {
	type Data struct {
		Role user.UserRole `json:"role"`
	}

	values := []struct {
		Data Data
		str  string
	}{
		{Data: Data{Role: user.AdminRole}, str: "admin"},
		{Data: Data{Role: user.DefaultRole}, str: "default"},
	}

	for _, v := range values {
		b, err := json.Marshal(v.Data)
		if err != nil {
			t.Error(err)
		}

		if !strings.Contains(string(b), v.str) {
			t.Error("Expected the user role to be encoded as a string")
		}

		var decoded Data
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Error(err)
		}

		if v.Data.Role != decoded.Role {
			t.Errorf("Expected the two roles match; actual: %d, recieved: %d", v.Data.Role, decoded.Role)
		}
	}

}
