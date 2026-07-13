package skylight

import "encoding/json"

// Collection preserves Skylight's full JSON document for command output while
// exposing decoded resource data to report and export callers.
type Collection[T any] struct {
	Data []T `json:"data"`
	raw  json.RawMessage
}

func (c Collection[T]) MarshalJSON() ([]byte, error) {
	if len(c.raw) != 0 {
		return c.raw, nil
	}
	type plain Collection[T]
	return json.Marshal(plain(c))
}

func decodeCollection[T any](raw []byte) (*Collection[T], error) {
	var out Collection[T]
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	out.raw = append(out.raw[:0], raw...)
	return &out, nil
}

// Document preserves Skylight's full JSON document while exposing one decoded
// resource to callers that need its identifier or attributes.
type Document[T any] struct {
	Data T `json:"data"`
	raw  json.RawMessage
}

func (d Document[T]) MarshalJSON() ([]byte, error) {
	if len(d.raw) != 0 {
		return d.raw, nil
	}
	type plain Document[T]
	return json.Marshal(plain(d))
}

func decodeDocument[T any](raw []byte) (*Document[T], error) {
	var out Document[T]
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	out.raw = append(out.raw[:0], raw...)
	return &out, nil
}
