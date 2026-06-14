package config

import "testing"

func TestLoadIdentityEnv(t *testing.T) {
	t.Setenv("HFR_LOGIN", "xatelitte")
	t.Setenv("HFR_PASSWD", "pw")
	t.Setenv("HFR_EXPECT_LOGIN", "xatelitte")
	t.Setenv("HFR_ALLOW_UNGUARDED_WRITES", "1")
	cfg := Load()
	if cfg.ExpectLogin != "xatelitte" {
		t.Fatalf("ExpectLogin = %q", cfg.ExpectLogin)
	}
	if !cfg.AllowUnguarded {
		t.Fatal("AllowUnguarded should be true")
	}
}
