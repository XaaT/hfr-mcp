package hfr

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// parseDoc is a helper that parses an HTML string into a goquery document.
func parseDoc(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse HTML: %v", err)
	}
	return doc
}

// TestParseNotifyStateCheckboxChecked: checkbox emaill with checked attr → true.
func TestParseNotifyStateCheckboxChecked(t *testing.T) {
	doc := parseDoc(t, `<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input class="checkbox opt" type="checkbox" value="1" name="emaill" checked="checked" id="emaill" />
<textarea name="content_form"></textarea>
</form></body></html>`)
	got := parseNotifyState(doc)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if !*got {
		t.Fatalf("expected true (subscribed), got false")
	}
}

// TestParseNotifyStateCheckboxUnchecked: checkbox emaill without checked → false.
func TestParseNotifyStateCheckboxUnchecked(t *testing.T) {
	doc := parseDoc(t, `<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input class="checkbox opt" type="checkbox" value="1" name="emaill" id="emaill" />
<textarea name="content_form"></textarea>
</form></body></html>`)
	got := parseNotifyState(doc)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if *got {
		t.Fatalf("expected false (not subscribed), got true")
	}
}

// TestParseNotifyStateHiddenOne: hidden emaill value="1" → true.
func TestParseNotifyStateHiddenOne(t *testing.T) {
	doc := parseDoc(t, `<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="hidden" name="emaill" value="1" />
<textarea name="content_form"></textarea>
</form></body></html>`)
	got := parseNotifyState(doc)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if !*got {
		t.Fatalf("expected true (hidden value=1), got false")
	}
}

// TestParseNotifyStateHiddenZero: hidden emaill value="0" → false.
func TestParseNotifyStateHiddenZero(t *testing.T) {
	doc := parseDoc(t, `<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="hidden" name="emaill" value="0" />
<textarea name="content_form"></textarea>
</form></body></html>`)
	got := parseNotifyState(doc)
	if got == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if *got {
		t.Fatalf("expected false (hidden value=0), got true")
	}
}

// TestParseNotifyStateAbsent: no emaill field → nil.
func TestParseNotifyStateAbsent(t *testing.T) {
	doc := parseDoc(t, `<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
<input type="hidden" name="other_field" value="42" />
<textarea name="content_form"></textarea>
</form></body></html>`)
	got := parseNotifyState(doc)
	if got != nil {
		t.Fatalf("expected nil (field absent), got %v", *got)
	}
}

// notifyModeServer builds a test server where message.php serves the given
// emaill HTML snippet, and bddpost.php captures the POST body.
func notifyModeServer(t *testing.T, emailllHTML string, gotPost *url.Values, postCount *int32) *httptest.Server {
	t.Helper()
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
		_, _ = w.Write([]byte(`<html><body>
<form action="/bddpost.php?config=hfr.inc" method="post">
` + emailllHTML + `
<textarea name="content_form"></textarea>
</form></body></html>`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(postCount, 1)
		_ = r.ParseForm()
		*gotPost = r.PostForm
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	return httptest.NewServer(mux)
}

// TestReplyNotifyKeepChecked: NotifyKeep + checked emaill checkbox → POST carries emaill=1.
func TestReplyNotifyKeepChecked(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	// message.php renders checked emaill checkbox (subscribed).
	srv := notifyModeServer(t,
		`<input type="checkbox" value="1" name="emaill" checked="checked" id="emaill" />`,
		&gotPost, &postCount)
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
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST, got %d", postCount)
	}
	// Checked checkbox is preserved → emaill=1 in POST.
	if got := gotPost.Get("emaill"); got != "1" {
		t.Fatalf("NotifyKeep+checked: emaill=%q in POST, want 1", got)
	}
	// ReplyResult.Subscribed == true (state from form).
	if res.Subscribed == nil || !*res.Subscribed {
		t.Fatalf("NotifyKeep+checked: Subscribed=%v, want true", res.Subscribed)
	}
}

// TestReplyNotifyKeepUnchecked: NotifyKeep + unchecked emaill → POST has NO emaill field.
func TestReplyNotifyKeepUnchecked(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	// message.php renders unchecked emaill checkbox (not subscribed).
	srv := notifyModeServer(t,
		`<input type="checkbox" value="1" name="emaill" id="emaill" />`,
		&gotPost, &postCount)
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
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST, got %d", postCount)
	}
	// Unchecked checkbox is not submitted by browser semantics → no emaill in POST.
	if _, present := gotPost["emaill"]; present {
		t.Fatalf("NotifyKeep+unchecked: emaill must not be in POST, got %v", gotPost["emaill"])
	}
	// ReplyResult.Subscribed == false (state from form).
	if res.Subscribed == nil || *res.Subscribed {
		t.Fatalf("NotifyKeep+unchecked: Subscribed=%v, want false", res.Subscribed)
	}
}

// TestReplyNotifySubscribe: NotifySubscribe + unchecked emaill → POST carries emaill=1;
// ReplyResult.Subscribed == true.
func TestReplyNotifySubscribe(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	// message.php renders UNCHECKED emaill — the mode must override it.
	srv := notifyModeServer(t,
		`<input type="checkbox" value="1" name="emaill" id="emaill" />`,
		&gotPost, &postCount)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	res, err := c.Reply(23, 35421, "hello", "id:1214571", NotifySubscribe)
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST, got %d", postCount)
	}
	// NotifySubscribe forces emaill=1 regardless of form state.
	if got := gotPost.Get("emaill"); got != "1" {
		t.Fatalf("NotifySubscribe: emaill=%q in POST, want 1", got)
	}
	// ReplyResult.Subscribed == true.
	if res.Subscribed == nil || !*res.Subscribed {
		t.Fatalf("NotifySubscribe: Subscribed=%v, want true", res.Subscribed)
	}
}

// TestReplyNotifyUnsubscribe: NotifyUnsubscribe + checked emaill → POST carries emaill=0;
// ReplyResult.Subscribed == false.
func TestReplyNotifyUnsubscribe(t *testing.T) {
	var gotPost url.Values
	var postCount int32
	// message.php renders CHECKED emaill — the mode must override it.
	srv := notifyModeServer(t,
		`<input type="checkbox" value="1" name="emaill" checked="checked" id="emaill" />`,
		&gotPost, &postCount)
	defer srv.Close()

	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	res, err := c.Reply(23, 35421, "hello", "id:1214571", NotifyUnsubscribe)
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if atomic.LoadInt32(&postCount) != 1 {
		t.Fatalf("expected one POST, got %d", postCount)
	}
	// NotifyUnsubscribe forces emaill=0 regardless of form state.
	if got := gotPost.Get("emaill"); got != "0" {
		t.Fatalf("NotifyUnsubscribe: emaill=%q in POST, want 0", got)
	}
	// ReplyResult.Subscribed == false.
	if res.Subscribed == nil || *res.Subscribed {
		t.Fatalf("NotifyUnsubscribe: Subscribed=%v, want false", res.Subscribed)
	}
}

func TestParseNotifyMode(t *testing.T) {
	cases := []struct {
		in      string
		want    NotifyMode
		wantErr bool
	}{
		{"", NotifyKeep, false},
		{"on", NotifySubscribe, false},
		{"off", NotifyUnsubscribe, false},
		{"OFF", NotifyKeep, true},
		{"true", NotifyKeep, true},
		{"1", NotifyKeep, true},
		{"keep", NotifyKeep, true},
	}
	for _, tc := range cases {
		got, err := ParseNotifyMode(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("ParseNotifyMode(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
		if !tc.wantErr && got != tc.want {
			t.Fatalf("ParseNotifyMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
