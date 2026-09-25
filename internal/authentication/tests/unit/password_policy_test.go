package unit_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/authentication"
)

func TestPasswordPolicyEnforcesOnlyLengthBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "short", password: "short", wantErr: authentication.ErrPasswordTooShort},
		{name: "passphrase", password: "correct horse battery staple"},
		{name: "unicode characters", password: strings.Repeat("界", 12)},
		{name: "long", password: strings.Repeat("a", 129), wantErr: authentication.ErrPasswordTooLong},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := authentication.ValidatePassword(test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidatePassword() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
