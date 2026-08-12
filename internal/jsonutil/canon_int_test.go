package jsonutil

import "testing"

func TestDecodeCanonicalObjectWithIntegerFields(t *testing.T) {
	type document struct {
		Equivalent int `json:"equivalent"`
		Scaled     int `json:"scaled"`
		TimeoutMS  int `json:"timeoutMS"`
	}

	raw := []byte(
		`{"timeoutMS":60000,"equivalent":6e4,"scaled":1.2e1}`,
	)

	canonical, err := CanonicalizeObject(raw, len(raw))
	if err != nil {
		t.Fatalf("CanonicalizeObject() error = %v", err)
	}

	const expected = `{"equivalent":60000,"scaled":12,"timeoutMS":60000}`
	if string(canonical) != expected {
		t.Fatalf(
			"CanonicalizeObject() = %s, want %s",
			canonical,
			expected,
		)
	}

	decoded, err := DecodeCanonicalObject[document](raw, len(raw))
	if err != nil {
		t.Fatalf("DecodeCanonicalObject() error = %v", err)
	}
	if decoded.Equivalent != 60000 ||
		decoded.Scaled != 12 ||
		decoded.TimeoutMS != 60000 {
		t.Fatalf("DecodeCanonicalObject() = %+v", decoded)
	}
}
