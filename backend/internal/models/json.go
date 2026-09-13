package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap is a string->string map persisted as jsonb (e.g. consumption labels).
type JSONMap map[string]string

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("JSONMap: unsupported scan type")
	}
	if len(b) == 0 {
		*m = JSONMap{}
		return nil
	}
	return json.Unmarshal(b, m)
}

// JSONStrings is a []string persisted as jsonb (e.g. amenities, image URLs).
type JSONStrings []string

func (m JSONStrings) Value() (driver.Value, error) {
	if m == nil {
		return "[]", nil
	}
	return json.Marshal(m)
}

// MarshalJSON guarantees API responses never expose a bare `null` for this
// type, even when the underlying slice is nil (nullable jsonb column, or a
// row written before a field existed) — a null here crashes any frontend
// code that assumes an array (see LocationsTab pros/cons, 2026-09-13).
func (m JSONStrings) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(m))
}

func (m *JSONStrings) Scan(src any) error {
	if src == nil {
		*m = JSONStrings{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("JSONStrings: unsupported scan type")
	}
	if len(b) == 0 {
		*m = JSONStrings{}
		return nil
	}
	return json.Unmarshal(b, m)
}

// JSONNum is a string->float64 map persisted as jsonb (e.g. quantity per consumption level).
type JSONNum map[string]float64

func (m JSONNum) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

func (m *JSONNum) Scan(src any) error {
	if src == nil {
		*m = JSONNum{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("JSONNum: unsupported scan type")
	}
	if len(b) == 0 {
		*m = JSONNum{}
		return nil
	}
	return json.Unmarshal(b, m)
}
