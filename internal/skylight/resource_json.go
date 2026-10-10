package skylight

import "encoding/json"

// Keep new upstream fields in read results without changing the public array shape.
func (c *Category) UnmarshalJSON(data []byte) error {
	type plain Category
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = Category(decoded)
	c.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (c Category) MarshalJSON() ([]byte, error) {
	if len(c.raw) != 0 {
		return c.raw, nil
	}
	type plain Category
	return json.Marshal(plain(c))
}

func (c *Chore) UnmarshalJSON(data []byte) error {
	type plain Chore
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = Chore(decoded)
	c.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (c Chore) MarshalJSON() ([]byte, error) {
	if len(c.raw) != 0 {
		return c.raw, nil
	}
	type plain Chore
	return json.Marshal(plain(c))
}

func (e *CalendarEvent) UnmarshalJSON(data []byte) error {
	type plain CalendarEvent
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*e = CalendarEvent(decoded)
	e.raw = append(json.RawMessage(nil), data...)
	return nil
}

func (e CalendarEvent) MarshalJSON() ([]byte, error) {
	type plain CalendarEvent
	if len(e.raw) == 0 {
		return json.Marshal(plain(e))
	}
	var original plain
	if err := json.Unmarshal(e.raw, &original); err != nil {
		return nil, err
	}
	if e.Attributes.StartsAt == original.Attributes.StartsAt && e.Attributes.EndsAt == original.Attributes.EndsAt {
		return e.raw, nil
	}
	// Weekly views change timestamp offsets. Keep all other upstream metadata.
	var resource map[string]json.RawMessage
	if err := json.Unmarshal(e.raw, &resource); err != nil {
		return nil, err
	}
	attributes := map[string]json.RawMessage{}
	if err := json.Unmarshal(resource["attributes"], &attributes); err != nil {
		return nil, err
	}
	if attributes == nil {
		attributes = map[string]json.RawMessage{}
	}
	if e.Attributes.StartsAt != original.Attributes.StartsAt {
		attributes["starts_at"], _ = json.Marshal(e.Attributes.StartsAt)
	}
	if e.Attributes.EndsAt != original.Attributes.EndsAt {
		attributes["ends_at"], _ = json.Marshal(e.Attributes.EndsAt)
	}
	encoded, err := json.Marshal(attributes)
	if err != nil {
		return nil, err
	}
	resource["attributes"] = encoded
	return json.Marshal(resource)
}
