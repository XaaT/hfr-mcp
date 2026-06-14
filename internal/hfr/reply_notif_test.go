package hfr

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

// replyNotifyServer logs in, serves a reply form on message.php that carries a
// CHECKED email-notify checkbox plus a few hidden state fields, and captures the
// body of the bddpost.php POST so the test can assert the notify field was
// round-tripped (bug #31). It also renders an UNCHECKED extra checkbox and a
// server hash_check that must NOT override the login-time token.
func replyNotifyServer(t *testing.T, pseudo, userID, notifyName, notifyVal string, gotPost *url.Values, postCount *int32) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: pseudo, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: userID, Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="LOGINHASH">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// A representative reply form. The notify checkbox is rendered checked,
		// meaning the user is currently subscribed; preserving it keeps the sub.
		_, _ = w.Write([]byte(`<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post" name="repondre">
<input type="hidden" name="hash_check" value="SERVERHASH">
<input type="hidden" name="cat" value="23">
<input type="hidden" name="post" value="35421">
<input type="hidden" name="stickold" value="99">
<input type="checkbox" name="` + notifyName + `" value="` + notifyVal + `" checked>
<input type="checkbox" name="some_other_opt" value="7">
<textarea name="content_form"></textarea>
<input type="submit" name="submit" value="Valider votre message">
</form>
</body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(postCount, 1)
		_ = r.ParseForm()
		*gotPost = r.PostForm
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	return httptest.NewServer(mux)
}

// Bug #31: a reply must carry the email-notify field that the reply form
// rendered as checked, so HFR keeps the user subscribed.
func TestReplyPreservesNotifyField(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	srv := replyNotifyServer(t, "xatelitte", "1214571", "notif", "1", &gotPost, &postCount)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}

	res, err := c.Reply(23, 35421, "hello", "id:1214571", NotifyKeep)
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if res.Identity.Pseudo != "xatelitte" {
		t.Fatalf("identity = %+v", res.Identity)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected exactly one POST, got %d", postCount)
	}

	// The notify checkbox was checked on the form -> it must be in the POST.
	if got := gotPost.Get("notif"); got != "1" {
		t.Fatalf("notify field missing/wrong in POST: notif=%q, full=%v", got, gotPost)
	}
	// An unchecked checkbox must NOT be sent (browser semantics).
	if _, present := gotPost["some_other_opt"]; present {
		t.Fatalf("unchecked checkbox must not be submitted, got %v", gotPost)
	}
	// The body is the client's, never the form's empty textarea.
	if got := gotPost.Get("content_form"); got != "hello" {
		t.Fatalf("content_form = %q, want %q", got, "hello")
	}
	// stickold is a field the base form deliberately controls (set to ""), so the
	// client value wins over the form's — preserved fields never clobber ours.
	if got := gotPost.Get("stickold"); got != "" {
		t.Fatalf("stickold = %q, want client-controlled empty value", got)
	}
	// hash_check must remain the login-time token, NOT the form's value.
	if got := gotPost.Get("hash_check"); got != "LOGINHASH" {
		t.Fatalf("hash_check = %q, want login token LOGINHASH (server form value must not override)", got)
	}
	// Client-controlled fields keep the client's value, not the form's.
	if got := gotPost.Get("post"); got != "35421" {
		t.Fatalf("post = %q, want 35421", got)
	}
}

// The notify field name is not hardcoded: whatever HFR calls the checked
// checkbox, it round-trips. This exercises the same path with a different name.
func TestReplyPreservesNotifyFieldUnderDifferentName(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	srv := replyNotifyServer(t, "xatelitte", "1214571", "mail_notif", "yes", &gotPost, &postCount)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetAllowUnguarded(true)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := c.Reply(23, 35421, "hello", "", NotifyKeep); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if got := gotPost.Get("mail_notif"); got != "yes" {
		t.Fatalf("notify field (alt name) missing in POST: %v", gotPost)
	}
}

// Guard interaction (#32): on an identity mismatch the reply form GET may run,
// but authenticatedPost must still refuse the POST. No bddpost.php call happens.
func TestReplyPreserveStillGuarded(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	srv := replyNotifyServer(t, "XaTriX", "54596", "notif", "1", &gotPost, &postCount)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("XaTriX", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := c.Reply(23, 35421, "hi", "", NotifyKeep); err == nil {
		t.Fatal("expected identity refusal")
	}
	if atomic.LoadInt32(&postCount) != 0 {
		t.Fatalf("POST must not be sent on mismatch, got %d", postCount)
	}
}

// Reinforcement test 1: TOCTOU cookie drift during the reply-form GET.
// The md_user cookie switches to XaTriX while the GET for message.php is
// served. authenticatedPost re-reads the live cookie before the POST and must
// refuse it. No bddpost.php call must be made.
func TestReplyPreserveCookieDriftBlocked(t *testing.T) {
	var postCount int32
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		// Log in as xatelitte.
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "xatelitte", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: "1214571", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="LOGINHASH">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// Session drifts: md_user flips to XaTriX while serving the reply form GET.
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "XaTriX", Path: "/"})
		_, _ = w.Write([]byte(`<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="checkbox" name="notif" value="1" checked>
<textarea name="content_form"></textarea>
</form></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&postCount, 1)
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	// After GET, cookie is XaTriX; guard re-checks before POST → must refuse.
	_, err := c.Reply(23, 35421, "hi", "", NotifyKeep)
	if err == nil {
		t.Fatal("expected identity refusal after cookie drift during form GET")
	}
	if atomic.LoadInt32(&postCount) != 0 {
		t.Fatalf("POST must not be sent after cookie drift, got %d", atomic.LoadInt32(&postCount))
	}
}

// Reinforcement test 1b: same TOCTOU drift but the guard is an id: constraint.
// The cached userID (1214571, from the xatelitte login) would still satisfy
// id:1214571 after the cookie drifts to another pseudo — unless currentIdentity
// invalidates it on pseudo drift. The POST must be refused, no bddpost.php call.
func TestReplyCookieDriftBlockedIdConstraint(t *testing.T) {
	var postCount int32
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "xatelitte", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: "1214571", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="LOGINHASH">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// Drift to another account (different pseudo) during the form GET.
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "XaTriX", Path: "/"})
		_, _ = w.Write([]byte(`<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="checkbox" name="notif" value="1" checked>
<textarea name="content_form"></textarea>
</form></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&postCount, 1)
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("id:1214571")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	_, err := c.Reply(23, 35421, "hi", "", NotifyKeep)
	if err == nil {
		t.Fatal("expected refusal: stale userID must not satisfy id: after pseudo drift")
	}
	if atomic.LoadInt32(&postCount) != 0 {
		t.Fatalf("POST must not be sent after drift, got %d", atomic.LoadInt32(&postCount))
	}
}

// Reinforcement test 2: if the reply-form GET fails or returns junk (no form),
// Reply must still POST (legacy fallback) rather than error out.
func TestReplyFallsBackWhenFormGetFails(t *testing.T) {
	var postCount int32
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "xatelitte", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: "1214571", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="LOGINHASH">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// Return junk HTML with no reply form — preserveReplyFormFields returns empty.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><p>Nothing here</p></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&postCount, 1)
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetAllowUnguarded(true)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := c.Reply(23, 35421, "hello", "", NotifyKeep); err != nil {
		t.Fatalf("reply must succeed despite junk form GET: %v", err)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST despite bad form GET, got %d", atomic.LoadInt32(&postCount))
	}
}

// Reinforcement test 3: client-controlled fields win over preserved form values.
// The reply form returns conflicting values for cat, sujet, and post; the POST
// must carry the client's values, not the form's.
func TestReplyClientFieldsWinOverPreserved(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: "xatelitte", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: "1214571", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="LOGINHASH">`))
	})
	mux.HandleFunc("/message.php", func(w http.ResponseWriter, r *http.Request) {
		// Form tries to inject conflicting values for client-controlled fields.
		_, _ = w.Write([]byte(`<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="hidden" name="hash_check" value="BADTOKEN">
<input type="hidden" name="cat" value="999">
<input type="hidden" name="post" value="00000">
<input type="hidden" name="sujet" value="INJECTED_SUBJECT">
<input type="checkbox" name="notif" value="1" checked>
<textarea name="content_form"></textarea>
</form></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&postCount, 1)
		_ = r.ParseForm()
		gotPost = r.PostForm
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetAllowUnguarded(true)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := c.Reply(23, 35421, "hello", "", NotifyKeep); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST, got %d", atomic.LoadInt32(&postCount))
	}

	// cat, post, sujet: client values must win.
	if got := gotPost.Get("cat"); got != "23" {
		t.Fatalf("cat = %q, want client value 23 (not form's 999)", got)
	}
	if got := gotPost.Get("post"); got != "35421" {
		t.Fatalf("post = %q, want client value 35421 (not form's 00000)", got)
	}
	if got := gotPost.Get("sujet"); got != "xatelitte" {
		t.Fatalf("sujet = %q, want pseudo xatelitte (not form's INJECTED_SUBJECT)", got)
	}
	// hash_check: login-time token must win, not BADTOKEN from form.
	if got := gotPost.Get("hash_check"); got != "LOGINHASH" {
		t.Fatalf("hash_check = %q, want LOGINHASH (not form's BADTOKEN)", got)
	}
	// The notify checkbox (not a client-set field) must still be preserved.
	if got := gotPost.Get("notif"); got != "1" {
		t.Fatalf("notif = %q, want 1 (should still be preserved)", got)
	}
}
