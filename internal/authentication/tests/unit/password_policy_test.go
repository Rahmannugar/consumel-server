package unit_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/authentication"
)

func TestPasswordPolicyEnforcesCredentialRequirements(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "short", password: "short", wantErr: authentication.ErrPasswordTooShort},
		{name: "valid", password: "Correct horse 7!"},
		{name: "unicode characters", password: "界界界界界界界界界A7!"},
		{name: "missing uppercase", password: "correct horse 7!", wantErr: authentication.ErrPasswordUppercase},
		{name: "missing number", password: "Correct horse!", wantErr: authentication.ErrPasswordNumber},
		{name: "missing special", password: "Correct horse 7", wantErr: authentication.ErrPasswordSpecial},
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
