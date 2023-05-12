package nullx

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"

	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/null/v8/convert"
	"go.infratographer.com/x/gidx"
)

// PrefixedID is a nullable PrefixedID. It supports SQL and JSON serialization.
type PrefixedID struct {
	PrefixedID gidx.PrefixedID
	Valid      bool
}

// PrefixedIDFrom creates a new PrefixedID that will never be blank.
func PrefixedIDFrom(s gidx.PrefixedID) PrefixedID {
	return NewPrefixedID(s, true)
}

// PrefixedIDFromPtr creates a new PrefixedID that be null if s is nil.
func PrefixedIDFromPtr(s *gidx.PrefixedID) PrefixedID {
	if s == nil {
		return NewPrefixedID("", false)
	}

	return NewPrefixedID(*s, true)
}

// NewPrefixedID creates a new PrefixedID
func NewPrefixedID(s gidx.PrefixedID, valid bool) PrefixedID {
	return PrefixedID{
		PrefixedID: s,
		Valid:      valid,
	}
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *PrefixedID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null.NullBytes) {
		s.PrefixedID = ""
		s.Valid = false

		return nil
	}

	if err := json.Unmarshal(data, &s.PrefixedID); err != nil {
		return err
	}

	s.Valid = true

	return nil
}

// MarshalJSON implements json.Marshaler.
func (s PrefixedID) MarshalJSON() ([]byte, error) {
	if !s.Valid {
		return null.NullBytes, nil
	}

	return json.Marshal(s.PrefixedID)
}

// MarshalText implements encoding.TextMarshaler.
func (s PrefixedID) MarshalText() ([]byte, error) {
	if !s.Valid {
		return []byte{}, nil
	}

	return []byte(s.PrefixedID), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *PrefixedID) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		s.Valid = false

		return nil
	}

	s.PrefixedID = gidx.PrefixedID(text)
	s.Valid = true

	return nil
}

// SetValid changes this PrefixedID's value and also sets it to be non-null.
func (s *PrefixedID) SetValid(v gidx.PrefixedID) {
	s.PrefixedID = v
	s.Valid = true
}

// Ptr returns a pointer to this PrefixedID's value, or a nil pointer if this PrefixedID is null.
func (s PrefixedID) Ptr() *gidx.PrefixedID {
	if !s.Valid {
		return nil
	}

	return &s.PrefixedID
}

// IsZero returns true for null ids, for potential future omitempty support.
func (s PrefixedID) IsZero() bool {
	return !s.Valid
}

// Scan implements the Scanner interface.
func (s *PrefixedID) Scan(value interface{}) error {
	if value == nil {
		s.PrefixedID, s.Valid = "", false

		return nil
	}

	s.Valid = true

	return convert.ConvertAssign(&s.PrefixedID, value)
}

// Value implements the driver Valuer interface.
func (s PrefixedID) Value() (driver.Value, error) {
	if !s.Valid {
		return nil, nil
	}

	return string(s.PrefixedID), nil
}
