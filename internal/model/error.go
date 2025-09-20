package model

import (
	"errors"
)

var (
	ErrDocContentTooLarge = errors.New("content too large (max 10MB)")
	ErrEmptyDocContent    = errors.New("empty doc content")
	ErrMessageTooLong     = errors.New("message too long (max 1000 characters)")
)
