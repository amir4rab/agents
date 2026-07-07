package domaintypes_test

import (
	"encoding/json"
	"fmt"
	"testing"

	domaintypes "github.com/nilafzar/agents/services/manager/internal/model/internal/types"
)

func TestOptional(t *testing.T) {
	type Payload struct {
		Name domaintypes.Optional[string] `json:"name"`
	}

	values := []struct {
		set     bool
		value   string
		payload string
	}{
		{true, "johndoe", "{\"name\":\"johndoe\"}"},
		{true, "", "{\"name\":null}"},
		{false, "", "{\"username\":\"johndoe\"}"},
	}

	for i, v := range values {
		t.Run(fmt.Sprintf("test %d", i+1), func(t *testing.T) {
			var payload Payload
			if err := json.Unmarshal([]byte(v.payload), &payload); err != nil {
				t.Error(err)
			}

			if payload.Name.Set != v.set {
				t.Error("Missmatch in set")
			}

			if payload.Name.Value == nil && v.value != "" {
				t.Error("Missmatch in value")
			}
		})
	}
}
