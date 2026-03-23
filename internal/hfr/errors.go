package hfr

import "fmt"

// HfrError represents a forum-specific error
type HfrError struct {
	Code    string // "auth", "flood", "rights", "locked", "hash", "parse"
	Message string
}

func (e *HfrError) Error() string {
	return fmt.Sprintf("hfr [%s]: %s", e.Code, e.Message)
}

var (
	ErrNotAuthenticated = &HfrError{Code: "auth", Message: "not authenticated"}
	ErrInvalidCreds     = &HfrError{Code: "auth", Message: "invalid credentials"}
	ErrSessionExpired   = &HfrError{Code: "auth", Message: "session expired"}
	ErrNoHashCheck      = &HfrError{Code: "hash", Message: "could not extract hash_check"}
	ErrFloodLimit       = &HfrError{Code: "flood", Message: "flood limit reached"}
	ErrNoRights         = &HfrError{Code: "rights", Message: "no rights to edit this message"}
	ErrTopicLocked      = &HfrError{Code: "locked", Message: "topic is locked"}
)
