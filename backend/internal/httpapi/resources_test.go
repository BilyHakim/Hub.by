package httpapi

import "testing"

func TestValidateTransferInput(t *testing.T) {
	tests := []struct {
		name  string
		input transferInput
		valid bool
	}{
		{
			name:  "valid transfer",
			input: transferInput{SourceAccountID: 1, DestinationAccountID: 2, Amount: 100000, OccurredAt: "2026-08-01"},
			valid: true,
		},
		{
			name:  "same account",
			input: transferInput{SourceAccountID: 1, DestinationAccountID: 1, Amount: 100000, OccurredAt: "2026-08-01"},
		},
		{
			name:  "non-positive amount",
			input: transferInput{SourceAccountID: 1, DestinationAccountID: 2, Amount: 0, OccurredAt: "2026-08-01"},
		},
		{
			name:  "invalid date",
			input: transferInput{SourceAccountID: 1, DestinationAccountID: 2, Amount: 100000, OccurredAt: "01-08-2026"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateTransferInput(test.input)
			if (err == nil) != test.valid {
				t.Fatalf("validateTransferInput() error = %v, valid = %v", err, test.valid)
			}
		})
	}
}
