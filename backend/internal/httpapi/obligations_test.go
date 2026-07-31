package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

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
