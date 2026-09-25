package authentication

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

const (
	minimumPasswordLength = 12
	maximumPasswordLength = 128
)

var (
	ErrPasswordTooShort  = errors.New("password must contain at least 12 characters")
	ErrPasswordTooLong   = errors.New("password must contain at most 128 characters")
	ErrPasswordUppercase = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNumber    = errors.New("password must contain at least one number")
	ErrPasswordSpecial   = errors.New("password must contain at least one special character")
)

func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minimumPasswordLength {
		return ErrPasswordTooShort
	}
	if length > maximumPasswordLength {
		return ErrPasswordTooLong
	}

	var hasUppercase, hasNumber, hasSpecial bool
	for _, character := range password {
		hasUppercase = hasUppercase || unicode.IsUpper(character)
		hasNumber = hasNumber || unicode.IsDigit(character)
		hasSpecial = hasSpecial || unicode.IsPunct(character) || unicode.IsSymbol(character)
	}
	if !hasUppercase {
		return ErrPasswordUppercase
	}
	if !hasNumber {
		return ErrPasswordNumber
	}
	if !hasSpecial {
		return ErrPasswordSpecial
	}
	return nil
}
