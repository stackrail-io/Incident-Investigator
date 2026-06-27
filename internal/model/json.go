package model

import "encoding/json"

// jsonMarshal is a tiny indirection so custom MarshalJSON implementations do not
// need to import encoding/json individually.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
