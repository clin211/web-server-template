package validation

import (
	"errors"
	"testing"
)

func TestValidateSelectedFields(t *testing.T) {
	type input struct {
		Value   *string
		Ignored string
		private string
	}
	value := "value"
	invalid := errors.New("invalid value")
	for _, test := range []struct {
		name           string
		input          any
		fields         []string
		validatorError error
		wantError      error
		wantCalls      int
	}{
		{name: "dereference pointer", input: &input{Value: &value}, fields: []string{"Value"}, wantCalls: 1},
		{name: "skip nil field", input: input{}, fields: []string{"Value"}},
		{name: "skip private and missing", input: input{Value: &value}, fields: []string{"private", "Missing", "Ignored"}},
		{name: "propagate validation error", input: input{Value: &value}, fields: []string{"Value"}, validatorError: invalid, wantError: invalid, wantCalls: 1},
		{name: "reject nil struct", input: (*input)(nil), wantError: errExpectedStruct},
		{name: "reject non struct", input: 1, wantError: errExpectedStruct},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			err := ValidateSelectedFields(test.input, Rules{"Value": func(got any) error {
				calls++
				if got != value {
					t.Errorf("validator value = %v, want %s", got, value)
				}
				return test.validatorError
			}}, test.fields...)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if calls != test.wantCalls {
				t.Errorf("calls = %d, want %d", calls, test.wantCalls)
			}
		})
	}
}
