package authentication

import (
	"errors"
	"unicode/utf8"
)

const (
	minimumPasswordLength = 12
	maximumPasswordLength = 128
)

var (
	ErrPasswordTooShort = errors.New("password must contain at least 12 characters")
	ErrPasswordTooLong  = errors.New("password must contain at most 128 characters")
)

func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minimumPasswordLength {
		return ErrPasswordTooShort
	}
	if length > maximumPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}
