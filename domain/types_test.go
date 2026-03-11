package domain

import "testing"

func TestEvaluateBullsCows(t *testing.T) {
	b, c, err := EvaluateBullsCows("0123", "0321")
	if err != nil {
		t.Fatal(err)
	}
	if b != 2 || c != 2 {
		t.Fatalf("got %d,%d", b, c)
	}
}

func TestValidateSecretCode(t *testing.T) {
	if err := ValidateSecretCode("0012"); err == nil {
		t.Fatal("expected duplicate digits error")
	}
	if err := ValidateSecretCode("0123"); err != nil {
		t.Fatal(err)
	}
}
