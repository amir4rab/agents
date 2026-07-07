package provider_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/provider"
)

func TestKindJSON(t *testing.T) {
	type Data struct {
		Kind provider.Kind `json:"kind"`
	}

	values := []struct {
		Data Data
		str  string
	}{
		{Data: Data{Kind: provider.Docker}, str: "docker"},
		{Data: Data{Kind: provider.Podman}, str: "podman"},
		{Data: Data{Kind: provider.Firecracker}, str: "firecracker"},
		{Data: Data{Kind: provider.QEMU}, str: "qemu"},
		{Data: Data{Kind: provider.Container}, str: "container"},
	}

	for _, v := range values {
		b, err := json.Marshal(v.Data)
		if err != nil {
			t.Error(err)
		}

		if !strings.Contains(string(b), v.str) {
			t.Error("Expected the Docker kind to be encoded as an string")
		}

		var decoded Data
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Error(err)
		}

		if v.Data.Kind != decoded.Kind {
			t.Errorf("Expected the two kinds match; actual: %d, recieved: %d", v.Data.Kind, decoded.Kind)
		}
	}

}
