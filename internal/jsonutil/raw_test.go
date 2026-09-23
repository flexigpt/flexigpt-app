package jsonutil

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSONStringRawIntoUnwrapsWailsJSONString(t *testing.T) {
	type args struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	type request struct {
		Args JSONRawString `json:"args"`
	}

	var requestValue request
	if err := json.Unmarshal(
		[]byte(`{"args":"{\"name\":\"Ada\",\"count\":2}"}`),
		&requestValue,
	); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	got, err := DecodeJSONStringRawInto[args](requestValue.Args)
	if err != nil {
		t.Fatalf("DecodeJSONStringRawInto() error = %v", err)
	}
	if got != (args{Name: "Ada", Count: 2}) {
		t.Fatalf("DecodeJSONStringRawInto() = %+v", got)
	}
}

func TestDecodeJSONStringRawAcceptsDirectRawJSON(t *testing.T) {
	raw, err := DecodeJSONStringRaw(
		JSONRawString(`{"enabled":true}`),
	)
	if err != nil {
		t.Fatalf("DecodeJSONStringRaw() error = %v", err)
	}

	const want = `{"enabled":true}`
	if string(raw) != want {
		t.Fatalf("DecodeJSONStringRaw() = %q, want %q", raw, want)
	}
}

func TestDecodeJSONStringRawRejectsInvalidJSON(t *testing.T) {
	_, err := DecodeJSONStringRaw(JSONRawString(`{"enabled":`))
	if err == nil {
		t.Fatal("DecodeJSONStringRaw() succeeded for invalid JSON")
	}
}
