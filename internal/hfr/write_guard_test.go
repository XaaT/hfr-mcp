package hfr

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// writeServer logs in as pseudo/userID and counts write POSTs (bddpost.php for
// reply/mp/new, bdd.php for edit). message.php serves a minimal edit page so
// Edit's pre-POST GET succeeds.
func writeServer(t *testing.T, pseudo, userID string, posted *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: pseudo, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: userID, Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="H">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(posted, 1)
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	mux.HandleFunc("/bdd.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(posted, 1)
		_, _ = w.Write([]byte("Message édité avec succès"))
	})
	return httptest.NewServer(mux)
}

func TestReplyRefusedOnMismatchNoPost(t *testing.T) {
	var posted int32
	srv := writeServer(t, "XaTriX", "54596", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("XaTriX", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	_, err := c.Reply(23, 35421, "hi", "")
	if err == nil {
		t.Fatal("expected identity refusal")
	}
	if atomic.LoadInt32(&posted) != 0 {
		t.Fatalf("POST must not be sent on mismatch, got %d", posted)
	}
}

func TestReplyAllowedReturnsIdentity(t *testing.T) {
	var posted int32
	srv := writeServer(t, "xatelitte", "1214571", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := c.Reply(23, 35421, "hi", "id:1214571")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if id.Pseudo != "xatelitte" || atomic.LoadInt32(&posted) != 1 {
		t.Fatalf("id=%+v posted=%d", id, atomic.LoadInt32(&posted))
	}
}

// Edit goes through the same guard; a mismatch must not reach the bdd.php POST,
// even though Edit does a GET on message.php first.
func TestEditRefusedOnMismatchNoPost(t *testing.T) {
	var posted int32
	srv := writeServer(t, "XaTriX", "54596", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("XaTriX", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	_, err := c.Edit(23, 35421, 74497677, "x", "")
	if err == nil {
		t.Fatal("expected identity refusal")
	}
	if atomic.LoadInt32(&posted) != 0 {
		t.Fatalf("edit POST must not be sent on mismatch, got %d", posted)
	}
}

// TOCTOU: the session drifts to another account during Edit's pre-POST GET.
// The guard re-reads the live cookie before the POST, so the edit is refused.
func TestEditRefusedOnCookieDriftNoPost(t *testing.T) {
	var posted int32
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "xatelitte", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: "1214571", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="H">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// Drift: the session flips to another account between the GET and the POST.
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "XaTriX", Path: "/"})
		_, _ = w.Write([]byte(`<html><body></body></html>`))
	})
	mux.HandleFunc("/bdd.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&posted, 1)
		_, _ = w.Write([]byte("Message édité avec succès"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	_, err := c.Edit(23, 35421, 74497677, "x", "")
	if err == nil {
		t.Fatal("expected refusal after mid-call cookie drift")
	}
	if atomic.LoadInt32(&posted) != 0 {
		t.Fatalf("edit POST must not be sent after drift, got %d", posted)
	}
}

// With no expected account configured but the explicit opt-out, a write goes through.
func TestReplyAllowedWhenUnguarded(t *testing.T) {
	var posted int32
	srv := writeServer(t, "xatelitte", "1214571", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetAllowUnguarded(true)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := c.Reply(23, 35421, "hi", "")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if id.Pseudo != "xatelitte" || atomic.LoadInt32(&posted) != 1 {
		t.Fatalf("id=%+v posted=%d", id, atomic.LoadInt32(&posted))
	}
}
