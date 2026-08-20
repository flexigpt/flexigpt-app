package overlay

import (
	"encoding/json"
	"slices"
)

type BundleID string

func (BundleID) Group() GroupID { return "bundles" }
func (b BundleID) ID() KeyID    { return KeyID(b) }

type TemplateID string

func (TemplateID) Group() GroupID { return "templates" }
func (t TemplateID) ID() KeyID    { return KeyID(t) }

type OtherID string

func (OtherID) Group() GroupID { return "other" }
func (o OtherID) ID() KeyID    { return KeyID(o) }

type CompositeKey struct{ A, B string }

func (CompositeKey) Group() GroupID { return "composite" }
func (c CompositeKey) ID() KeyID    { return KeyID(c.A + "::" + c.B) }

type DuplicateBundleID string

func (DuplicateBundleID) Group() GroupID { return "bundles" }
func (d DuplicateBundleID) ID() KeyID    { return KeyID(d) }

func marshalBool(b bool) json.RawMessage {
	if b {
		return json.RawMessage("true")
	}
	return json.RawMessage("false")
}

func containsErr(err error, substr string) bool {
	return err != nil && substr != "" && func() bool {
		return (len(err.Error()) >= len(substr)) && func() bool {
			for i := range err.Error() {
				if len(err.Error())-i < len(substr) {
					return false
				}
				if err.Error()[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()
	}()
}

func containsString(list []string, s string) bool {
	return slices.Contains(list, s)
}

func equalIntSlice(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringIntMap(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
