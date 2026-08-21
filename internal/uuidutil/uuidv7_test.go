package uuidutil

import (
	"errors"
	"testing"
	"time"
	"uuid"
)

func TestNewUUIDv7(t *testing.T) {
	t.Run("generate and validate id", func(t *testing.T) {
		before := time.Now().UTC()
		id := NewUUIDv7()
		after := time.Now().UTC()

		if err := ValidateUUIDv7(id); err != nil {
			t.Fatalf("ValidateUUIDv7(%q) error = %v, want nil", id, err)
		}

		gotTime, err := GetUUIDv7UnixTime(id)
		if err != nil {
			t.Fatalf("GetUUIDv7UnixTime(%q) error = %v, want nil", id, err)
		}

		if gotTime.Location() != time.UTC {
			t.Errorf("GetUUIDv7UnixTime(%q) location = %v, want UTC", id, gotTime.Location())
		}

		// UUIDv7 timestamps have millisecond precision. The one-second
		// window avoids flakes from scheduling or clock precision.
		lowerBound := before.Add(-time.Second)
		upperBound := after.Add(time.Second)

		if gotTime.Before(lowerBound) || gotTime.After(upperBound) {
			t.Errorf(
				"GetUUIDv7UnixTime(%q) = %v, want time between %v and %v",
				id,
				gotTime,
				lowerBound,
				upperBound,
			)
		}
	})
}

func TestValidateUUIDv7(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		wantErr        bool
		wantErrInvalid bool
		wantMessage    string
	}{
		{
			name:  "valid v7 UUID",
			value: "018bcfe5-687b-7000-8000-000000000000",
		},
		{
			name:        "v4 UUID is rejected",
			value:       "018bcfe5-687b-4000-8000-000000000000",
			wantErr:     true,
			wantMessage: "bad uuid version: 4",
		},
		{
			name:           "malformed UUID is rejected",
			value:          "not-a-uuid",
			wantErr:        true,
			wantErrInvalid: true,
		},
		{
			name:           "empty UUID is rejected",
			value:          "",
			wantErr:        true,
			wantErrInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUIDv7(tt.value)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUUIDv7(%q) error = %v, want error = %v", tt.value, err, tt.wantErr)
			}

			if errors.Is(err, errInvalid) != tt.wantErrInvalid {
				t.Errorf(
					"ValidateUUIDv7(%q) errors.Is(err, errInvalid) = %v, want %v; error = %v",
					tt.value,
					errors.Is(err, errInvalid),
					tt.wantErrInvalid,
					err,
				)
			}

			if tt.wantMessage != "" && err != nil && err.Error() != tt.wantMessage {
				t.Errorf(
					"ValidateUUIDv7(%q) error = %q, want %q",
					tt.value,
					err.Error(),
					tt.wantMessage,
				)
			}
		})
	}
}

func TestGetUUIDv7UnixTime(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		want           time.Time
		wantErr        bool
		wantErrInvalid bool
	}{
		{
			name: "one millisecond after Unix epoch",
			id:   "00000000-0001-7000-8000-000000000000",
			want: time.Unix(0, 1_000_000).UTC(),
		},
		{
			name: "known timestamp",
			id:   "018bcfe5-687b-7000-8000-000000000000",
			want: time.Unix(1_700_000_000, 123_000_000).UTC(),
		},
		{
			name:           "non v7 UUID is rejected",
			id:             "018bcfe5-687b-4000-8000-000000000000",
			wantErr:        true,
			wantErrInvalid: true,
		},
		{
			name:           "malformed UUID is rejected",
			id:             "invalid",
			wantErr:        true,
			wantErrInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetUUIDv7UnixTime(tt.id)

			if (err != nil) != tt.wantErr {
				t.Fatalf("GetUUIDv7UnixTime(%q) error = %v, want error = %v", tt.id, err, tt.wantErr)
			}

			if errors.Is(err, errInvalid) != tt.wantErrInvalid {
				t.Errorf(
					"GetUUIDv7UnixTime(%q) errors.Is(err, errInvalid) = %v, want %v; error = %v",
					tt.id,
					errors.Is(err, errInvalid),
					tt.wantErrInvalid,
					err,
				)
			}

			if tt.wantErr {
				if !got.IsZero() {
					t.Errorf("GetUUIDv7UnixTime(%q) time = %v, want zero time on error", tt.id, got)
				}
				return
			}

			if !got.Equal(tt.want) {
				t.Errorf("GetUUIDv7UnixTime(%q) = %v, want %v", tt.id, got, tt.want)
			}

			if got.Location() != time.UTC {
				t.Errorf("GetUUIDv7UnixTime(%q) location = %v, want UTC", tt.id, got.Location())
			}
		})
	}
}

func TestGetv7Time(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		wantSec        int64
		wantNSec       int64
		wantErr        bool
		wantErrInvalid bool
	}{
		{
			name:     "one millisecond after Unix epoch",
			id:       "00000000-0001-7000-8000-000000000000",
			wantSec:  0,
			wantNSec: 1_000_000,
		},
		{
			name:     "known timestamp",
			id:       "018bcfe5-687b-7000-8000-000000000000",
			wantSec:  1_700_000_000,
			wantNSec: 123_000_000,
		},
		{
			name:           "non v7 UUID is rejected",
			id:             "018bcfe5-687b-4000-8000-000000000000",
			wantErr:        true,
			wantErrInvalid: true,
		},
		{
			name:           "malformed UUID is rejected",
			id:             "invalid",
			wantErr:        true,
			wantErrInvalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSec, gotNSec, err := getv7Time(tt.id)

			if (err != nil) != tt.wantErr {
				t.Fatalf("getv7Time(%q) error = %v, want error = %v", tt.id, err, tt.wantErr)
			}

			if errors.Is(err, errInvalid) != tt.wantErrInvalid {
				t.Errorf(
					"getv7Time(%q) errors.Is(err, errInvalid) = %v, want %v; error = %v",
					tt.id,
					errors.Is(err, errInvalid),
					tt.wantErrInvalid,
					err,
				)
			}

			if tt.wantErr {
				if gotSec != 0 || gotNSec != 0 {
					t.Errorf(
						"getv7Time(%q) = (%d, %d), want (0, 0) on error",
						tt.id,
						gotSec,
						gotNSec,
					)
				}
				return
			}

			if gotSec != tt.wantSec || gotNSec != tt.wantNSec {
				t.Errorf(
					"getv7Time(%q) = (%d, %d), want (%d, %d)",
					tt.id,
					gotSec,
					gotNSec,
					tt.wantSec,
					tt.wantNSec,
				)
			}
		})
	}
}

func TestIsUUIDv7(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantErr     bool
		wantMessage string
	}{
		{
			name:  "version 7 UUID",
			value: "018bcfe5-687b-7000-8000-000000000000",
		},
		{
			name:        "version 4 UUID",
			value:       "018bcfe5-687b-4000-8000-000000000000",
			wantErr:     true,
			wantMessage: "bad uuid version: 4",
		},
		{
			name:        "version 0 UUID",
			value:       "018bcfe5-687b-0000-8000-000000000000",
			wantErr:     true,
			wantMessage: "bad uuid version: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := uuid.Parse(tt.value)
			if err != nil {
				t.Fatalf("uuid.Parse(%q) error = %v", tt.value, err)
			}

			err = isUUIDv7(id)

			if (err != nil) != tt.wantErr {
				t.Fatalf("isUUIDv7(%q) error = %v, want error = %v", tt.value, err, tt.wantErr)
			}

			if tt.wantMessage != "" && err != nil && err.Error() != tt.wantMessage {
				t.Errorf("isUUIDv7(%q) error = %q, want %q", tt.value, err.Error(), tt.wantMessage)
			}
		})
	}
}
