package domaintypes

import (
	"encoding/json"
)

// Optional field is designed to be used for detecting if an optional field is truly set to null
// Its behavior are as follows:
//   - The field has a value: Field will be populated and the `set` field will be true.
//   - The field has a nil value: The `set` field will be true, which means the co-responding optional value must be nil.
//   - The field isn't present: The `set` field will be false and the value must not be updated.
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(b []byte) error {
	o.Set = true

	if string(b) == "null" {
		o.Value = nil
		return nil
	}

	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	o.Value = &v
	return nil
}
