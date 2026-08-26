package options

import "testing"

func TestLoginSecurityOptionsValidate(t *testing.T) {
	options := NewLoginSecurityOptions()
	if errs := options.Validate(); len(errs) != 0 {
		t.Fatalf("default Validate() errors = %v", errs)
	}
	options.MaxAttempts = 0
	if errs := options.Validate(); len(errs) == 0 {
		t.Fatal("Validate() returned no error for zero max attempts")
	}
	options.Enabled = false
	if errs := options.Validate(); len(errs) != 0 {
		t.Fatalf("disabled Validate() errors = %v", errs)
	}
}
