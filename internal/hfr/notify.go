package hfr

// NotifyMode controls the email-notification subscription for a topic when
// posting a reply. The default (NotifyKeep) preserves the current server-side
// state by round-tripping the value from the reply form.
type NotifyMode int

const (
	// NotifyKeep preserves the current email-notify subscription state.
	// The value round-tripped from the reply form is used as-is.
	NotifyKeep NotifyMode = iota
	// NotifySubscribe forces email notifications on (emaill=1).
	NotifySubscribe
	// NotifyUnsubscribe forces email notifications off (emaill=0).
	NotifyUnsubscribe
)

// ReplyResult holds the outcome of a successful Reply call.
type ReplyResult struct {
	// Identity is the account that posted the message, as verified by the
	// fail-closed guard immediately before the POST.
	Identity Identity
	// Subscribed is the email-notification state in effect after the post.
	// true = subscribed, false = unsubscribed, nil = state unknown (form fetch
	// failed or emaill field was not found).
	Subscribed *bool
}
