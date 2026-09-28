package models

import "errors"

var (
	ErrProjectEnvironmentUnavailable = errors.New("project environment is unavailable")
	ErrProjectEnvironmentInactive    = errors.New("project environment is inactive")
	ErrActiveAPIKeyExists            = errors.New("an active API key already exists")
	ErrActiveAPIKeyNotFound          = errors.New("an active API key was not found")
)
