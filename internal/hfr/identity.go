package hfr

import (
	"fmt"
	"strings"
)

// Identity is the account currently behind the session.
type Identity struct {
	Pseudo string
	UserID string
	// Authenticated is set by currentIdentity; false when constructed directly.
	Authenticated bool
}

// normalize folds case and trims surrounding space for pseudo comparison.
func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// isAllDigits reports whether s is a non-empty string of ASCII decimal digits.
// The empty check is load-bearing: it stops an empty constraint from being read
// as a numeric userId.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseExpect splits a typed constraint.
// "id:123" -> (true,"123"); "pseudo:xat" -> (false,"xat");
// bare all-digits -> (true, digits); otherwise (false, trimmed).
func parseExpect(want string) (isID bool, value string) {
	w := strings.TrimSpace(want)
	switch {
	case strings.HasPrefix(w, "id:"):
		return true, strings.TrimSpace(strings.TrimPrefix(w, "id:"))
	case strings.HasPrefix(w, "pseudo:"):
		return false, strings.TrimSpace(strings.TrimPrefix(w, "pseudo:"))
	default:
		if isAllDigits(w) {
			return true, w
		}
		return false, w
	}
}

// checkIdentity enforces the expected-account constraints against id.
// Fail-closed: with no constraint and no opt-out it refuses.
// The caller must hold c.mu (it runs inside the guarded write path); it does
// not lock itself, to stay reentrant-safe under authenticatedPost.
func (c *Client) checkIdentity(id Identity, expect string) error {
	var constraints []string
	if c.expectedLogin != "" {
		constraints = append(constraints, c.expectedLogin)
	}
	if expect != "" {
		constraints = append(constraints, expect)
	}
	if len(constraints) == 0 {
		if c.allowUnguarded {
			return nil
		}
		return ErrNoExpectedAccount
	}
	for _, want := range constraints {
		ok, err := identityMatches(id, want)
		if err != nil {
			return err
		}
		if !ok {
			return &HfrError{Code: "identity", Message: fmt.Sprintf(
				"write refused: connected as %q (userId %s) != expected %q", id.Pseudo, id.UserID, want)}
		}
	}
	return nil
}

// identityMatches reports whether id satisfies the constraint want.
// It errors if want targets a userId the session could not resolve.
func identityMatches(id Identity, want string) (bool, error) {
	isID, val := parseExpect(want)
	if isID {
		if id.UserID == "" {
			return false, &HfrError{Code: "identity", Message: "expected account is a userId but the session userId could not be resolved"}
		}
		return val == id.UserID, nil
	}
	return normalize(val) == normalize(id.Pseudo), nil
}
