package hfr

import "testing"

func TestCheckIdentity(t *testing.T) {
	id := Identity{Pseudo: "xatelitte", UserID: "1214571", Authenticated: true}

	t.Run("no constraint, fail-closed", func(t *testing.T) {
		c := &Client{}
		if err := c.checkIdentity(id, ""); err != ErrNoExpectedAccount {
			t.Fatalf("got %v, want ErrNoExpectedAccount", err)
		}
	})
	t.Run("no constraint, opt-out allows", func(t *testing.T) {
		c := &Client{allowUnguarded: true}
		if err := c.checkIdentity(id, ""); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})
	t.Run("server constraint match", func(t *testing.T) {
		c := &Client{expectedLogin: "xatelitte"}
		if err := c.checkIdentity(id, ""); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})
	t.Run("server constraint mismatch", func(t *testing.T) {
		c := &Client{expectedLogin: "XaTriX"}
		err := c.checkIdentity(id, "")
		if he, ok := err.(*HfrError); !ok || he.Code != "identity" {
			t.Fatalf("got %v, want identity error", err)
		}
	})
	t.Run("all constraints must pass: server match + call mismatch", func(t *testing.T) {
		c := &Client{expectedLogin: "xatelitte"}
		if err := c.checkIdentity(id, "id:54596"); err == nil {
			t.Fatal("expected mismatch error")
		}
	})
	t.Run("opt-out does not bypass an active constraint", func(t *testing.T) {
		c := &Client{allowUnguarded: true, expectedLogin: "XaTriX"}
		if err := c.checkIdentity(id, ""); err == nil {
			t.Fatal("expected identity error despite allowUnguarded")
		}
	})
}

func TestIdentityMatches(t *testing.T) {
	tests := []struct {
		name    string
		id      Identity
		want    string
		ok      bool
		wantErr bool
	}{
		{"pseudo exact", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "xatelitte", true, false},
		{"pseudo case+space", Identity{Pseudo: "xatelitte"}, "  XaTeLitte ", true, false},
		{"pseudo mismatch", Identity{Pseudo: "XaTriX"}, "xatelitte", false, false},
		{"pseudo prefix", Identity{Pseudo: "xatelitte"}, "pseudo:xatelitte", true, false},
		{"id prefix match", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "id:1214571", true, false},
		{"id prefix mismatch", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "id:54596", false, false},
		{"bare numeric -> userId", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "1214571", true, false},
		{"numeric pseudo via prefix", Identity{Pseudo: "1234"}, "pseudo:1234", true, false},
		{"id wanted but unresolved", Identity{Pseudo: "xatelitte", UserID: ""}, "id:1214571", false, true},
		{"zero is a real constraint", Identity{Pseudo: "x", UserID: "5"}, "0", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := identityMatches(tc.id, tc.want)
			if tc.wantErr {
				he, isHfr := err.(*HfrError)
				if !isHfr || he.Code != "identity" {
					t.Fatalf("err = %v, want *HfrError code=identity", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err = %v", err)
			}
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}
