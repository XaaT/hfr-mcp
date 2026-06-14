package hfr

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient points a Client at a test server.
func newTestClient(baseURL string) *Client {
	c := NewClient()
	c.baseURL = baseURL
	return c
}

// loginMux serves a minimal HFR login + profile. cookieUserID controls the
// md_user_id cookie (primary userId source); pageUserID controls the user=NNNN
// profile link in editprofil (fallback source). Either may be empty.
func loginMux(t *testing.T, pseudo, cookieUserID, pageUserID string, hashOK bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: pseudo, Path: "/"})
		if cookieUserID != "" {
			http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: cookieUserID, Path: "/"})
		}
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		if !hashOK {
			_, _ = w.Write([]byte("<html><body>no token</body></html>"))
			return
		}
		page := `<html><body><input type="hidden" name="hash_check" value="HASH123">`
		if pageUserID != "" {
			page += `<a href="/user/profil.php?config=hfr.inc&amp;user=` + pageUserID + `">profil</a>`
		}
		page += `</body></html>`
		_, _ = w.Write([]byte(page))
	})
	return httptest.NewServer(mux)
}

func TestLoginResolvesIdentity(t *testing.T) {
	srv := loginMux(t, "xatelitte", "1214571", "1214571", true)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := c.currentIdentity()
	if err != nil {
		t.Fatalf("currentIdentity: %v", err)
	}
	if id.Pseudo != "xatelitte" || id.UserID != "1214571" || !id.Authenticated {
		t.Fatalf("identity = %+v", id)
	}
}

// userId resolves from the md_user_id cookie alone (no profile link).
func TestLoginUserIDFromCookie(t *testing.T) {
	srv := loginMux(t, "xatelitte", "1214571", "", true)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if c.userID != "1214571" {
		t.Fatalf("userID = %q, want 1214571 (cookie path)", c.userID)
	}
}

// userId resolves from the editprofil page link when no cookie is present.
func TestLoginUserIDFromPageFallback(t *testing.T) {
	srv := loginMux(t, "xatelitte", "", "1214571", true)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if c.userID != "1214571" {
		t.Fatalf("userID = %q, want 1214571 (page fallback)", c.userID)
	}
}

func TestLoginRollbackOnHashFailure(t *testing.T) {
	srv := loginMux(t, "xatelitte", "1214571", "", false)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err == nil {
		t.Fatal("expected login error when hash_check missing")
	}
	if c.authed {
		t.Fatal("authed must stay false after failed login")
	}
}

// The server logs in a different account than requested: login must fail.
func TestLoginMismatchedCookie(t *testing.T) {
	srv := loginMux(t, "someoneelse", "54596", "", true)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err == nil {
		t.Fatal("expected login error when md_user cookie != requested pseudo")
	}
	if c.authed {
		t.Fatal("authed must stay false after mismatched login")
	}
}
