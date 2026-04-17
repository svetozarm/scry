package provider

import "errors"

var (
	ErrAuth            = errors.New("authentication or authorisation error")
	ErrRateLimit       = errors.New("rate limit exceeded")
	ErrTimeout         = errors.New("request timed out")
	ErrModelNotFound   = errors.New("model not found")
	ErrUnknownProvider = errors.New("unknown provider")
)
