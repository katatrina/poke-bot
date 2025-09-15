package types

import (
	"errors"
)

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrInvalidConfig  = errors.New("invalid config format")
	ErrServerFailed   = errors.New("server failed to start")
)
