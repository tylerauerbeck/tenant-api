package nullx

import (
	"encoding/json"
	"testing"

	"go.infratographer.com/x/gidx"
)

var (
	stringJSON          = []byte(`"test"`)
	blankPrefixedIDJSON = []byte(`""`)

	nullJSON    = []byte(`null`)
	invalidJSON = []byte(`:)`)

	boolJSON = []byte(`true`)
)

func TestPrefixedIDFrom(t *testing.T) {
	str := PrefixedIDFrom("test")

	assertStr(t, str, "PrefixedIDFrom() string")

	zero := PrefixedIDFrom("")
	if !zero.Valid {
		t.Error("PrefixedIDFrom(0)", "is invalid, but should be valid")
	}
}

func TestPrefixedIDFromPtr(t *testing.T) {
	s := gidx.PrefixedID("test")
	sptr := &s
	str := PrefixedIDFromPtr(sptr)

	assertStr(t, str, "PrefixedIDFromPtr() string")

	null := PrefixedIDFromPtr(nil)

	assertNullStr(t, null, "PrefixedIDFromPtr(nil)")
}

func TestUnmarshalPrefixedID(t *testing.T) {
	var str PrefixedID

	err := json.Unmarshal(stringJSON, &str)

	maybePanic(err)
	assertStr(t, str, "string json")

	var blank PrefixedID

	err = json.Unmarshal(blankPrefixedIDJSON, &blank)

	maybePanic(err)

	if !blank.Valid {
		t.Error("blank string should be valid")
	}

	var null PrefixedID
	err = json.Unmarshal(nullJSON, &null)

	maybePanic(err)
	assertNullStr(t, null, "null json")

	var badType PrefixedID
	err = json.Unmarshal(boolJSON, &badType)

	if err == nil {
		panic("err should not be nil")
	}

	assertNullStr(t, badType, "wrong type json")

	var invalid PrefixedID

	err = invalid.UnmarshalJSON(invalidJSON)

	if _, ok := err.(*json.SyntaxError); !ok {
		t.Errorf("expected json.SyntaxError, not %T", err)
	}

	assertNullStr(t, invalid, "invalid json")
}

func TestTextUnmarshalPrefixedID(t *testing.T) {
	var str PrefixedID

	err := str.UnmarshalText([]byte("test"))

	maybePanic(err)
	assertStr(t, str, "UnmarshalText() string")

	var null PrefixedID

	err = null.UnmarshalText([]byte(""))

	maybePanic(err)
	assertNullStr(t, null, "UnmarshalText() empty string")
}

func TestMarshalPrefixedID(t *testing.T) {
	str := PrefixedIDFrom("test")
	data, err := json.Marshal(str)

	maybePanic(err)
	assertJSONEquals(t, data, `"test"`, "non-empty json marshal")

	data, err = str.MarshalText()

	maybePanic(err)
	assertJSONEquals(t, data, "test", "non-empty text marshal")

	// empty values should be encoded as an empty string
	zero := PrefixedIDFrom("")
	data, err = json.Marshal(zero)

	maybePanic(err)
	assertJSONEquals(t, data, `""`, "empty json marshal")

	data, err = zero.MarshalText()

	maybePanic(err)
	assertJSONEquals(t, data, "", "string marshal text")

	null := PrefixedIDFromPtr(nil)
	data, err = json.Marshal(null)

	maybePanic(err)
	assertJSONEquals(t, data, `null`, "null json marshal")

	data, err = null.MarshalText()

	maybePanic(err)
	assertJSONEquals(t, data, "", "string marshal text")
}

func TestPrefixedIDPointer(t *testing.T) {
	str := PrefixedIDFrom("test")
	ptr := str.Ptr()

	if *ptr != "test" {
		t.Errorf("bad %s string: %#v ≠ %s\n", "pointer", ptr, "test")
	}

	null := NewPrefixedID("", false)
	ptr = null.Ptr()

	if ptr != nil {
		t.Errorf("bad %s string: %#v ≠ %s\n", "nil pointer", ptr, "nil")
	}
}

func TestPrefixedIDIsZero(t *testing.T) {
	str := PrefixedIDFrom("test")
	if str.IsZero() {
		t.Errorf("IsZero() should be false")
	}

	blank := PrefixedIDFrom("")
	if blank.IsZero() {
		t.Errorf("IsZero() should be false")
	}

	empty := NewPrefixedID("", true)
	if empty.IsZero() {
		t.Errorf("IsZero() should be false")
	}

	null := PrefixedIDFromPtr(nil)
	if !null.IsZero() {
		t.Errorf("IsZero() should be true")
	}
}

func TestPrefixedIDSetValid(t *testing.T) {
	change := NewPrefixedID("", false)

	assertNullStr(t, change, "SetValid()")
	change.SetValid("test")

	assertStr(t, change, "SetValid()")
}

func TestPrefixedIDScan(t *testing.T) {
	var str PrefixedID
	err := str.Scan("test")
	maybePanic(err)

	assertStr(t, str, "scanned string")

	var null PrefixedID
	err = null.Scan(nil)

	maybePanic(err)
	assertNullStr(t, null, "scanned null")
}

func maybePanic(err error) {
	if err != nil {
		panic(err)
	}
}

func assertStr(t *testing.T, s PrefixedID, from string) {
	if s.PrefixedID != "test" {
		t.Errorf("bad %s string: %s ≠ %s\n", from, s.PrefixedID, "test")
	}

	if !s.Valid {
		t.Error(from, "is invalid, but should be valid")
	}
}

func assertNullStr(t *testing.T, s PrefixedID, from string) {
	if s.Valid {
		t.Error(from, "is valid, but should be invalid")
	}
}

func assertJSONEquals(t *testing.T, data []byte, cmp string, from string) {
	if string(data) != cmp {
		t.Errorf("bad %s data: %s ≠ %s\n", from, data, cmp)
	}
}
