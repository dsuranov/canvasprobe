package figma

import (
	"errors"
	"fmt"
)

var (
	ErrUnauthorized     = errors.New("unauthorized (check token)")
	ErrForbidden        = errors.New("forbidden (check token scopes and file permissions)")
	ErrNotFound         = errors.New("file or resource not found")
	ErrRateLimited      = errors.New("rate limited after retry")
	ErrResponseTooLarge = errors.New("response exceeds size limit")
)

type APIError struct {
	Status    int
	Message   string
	Body      string
	RequestID string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("upstream API: %d %s", e.Status, e.Message)
}
