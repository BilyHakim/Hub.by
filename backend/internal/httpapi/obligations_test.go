package httpapi

import (
	"math"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentSources(t *testing.T) {
	tests := []struct {
		name  string
		input obligationPaymentInput
		valid bool
	}{
		{"legacy single wallet", obligationPaymentInput{Amount: 100, AccountID: 1}, true},
		{"split wallets", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 60}, {2, 40}}}, true},
		{"missing wallet", obligationPaymentInput{Amount: 100}, false},
		{"zero total", obligationPaymentInput{AccountID: 1}, false},
		{"duplicate wallets", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 60}, {1, 40}}}, false},
		{"underallocated", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 60}, {2, 30}}}, false},
		{"overallocated", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 60}, {2, 50}}}, false},
		{"negative source", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 110}, {2, -10}}}, false},
		{"zero source", obligationPaymentInput{Amount: 100, Sources: []paymentSource{{1, 100}, {2, 0}}}, false},
		{"ambiguous format", obligationPaymentInput{Amount: 100, AccountID: 1, Sources: []paymentSource{{2, 100}}}, false},
		{"overflow", obligationPaymentInput{Amount: math.MaxInt64, Sources: []paymentSource{{1, math.MaxInt64}, {2, math.MaxInt64}, {3, 1}}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources, err := test.input.paymentSources()
			if (err == nil) != test.valid {
				t.Fatalf("sources=%v error=%v, valid=%v", sources, err, test.valid)
			}
			if test.valid {
				var total int64
				for _, source := range sources {
					total += source.Amount
				}
				if total != test.input.Amount {
					t.Fatalf("total=%d want=%d", total, test.input.Amount)
				}
			}
		})
	}
}

func TestDecodeObligationInput(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/v1/obligations", strings.NewReader(`{
		"type":"debt",
		"name":"  PayLater laptop  ",
		"platform":"  Marketplace  ",
		"originalAmount":12000000,
		"installmentCount":12,
		"startDate":"2026-08-01",
		"notes":" test "
	}`))
	var input obligationInput
	start, ok := decodeObligationInput(request, &input)
	if !ok {
		t.Fatal("expected valid obligation")
	}
	if input.Name != "PayLater laptop" || input.Platform != "Marketplace" || input.Notes != "test" {
		t.Fatalf("input was not normalized: %+v", input)
	}
	if got := start.Format("2006-01-02"); got != "2026-08-01" {
		t.Fatalf("start date = %s", got)
	}
}

func TestDecodeObligationInputRejectsInvalidTenor(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/v1/obligations", strings.NewReader(`{
		"type":"debt","name":"PayLater","originalAmount":1000,
		"installmentCount":0,"startDate":"2026-08-01"
	}`))
	var input obligationInput
	if _, ok := decodeObligationInput(request, &input); ok {
		t.Fatal("expected invalid tenor to be rejected")
	}
}
