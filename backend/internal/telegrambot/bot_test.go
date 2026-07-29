package telegrambot

import "testing"

func TestParseAmount(t *testing.T) {
	tests := map[string]int64{
		"25000":    25_000,
		"25.000":   25_000,
		"Rp25.000": 25_000,
		"25rb":     25_000,
		"1,5jt":    1_500_000,
		"2juta":    2_000_000,
	}
	for input, expected := range tests {
		actual, err := parseAmount(input)
		if err != nil {
			t.Fatalf("parseAmount(%q): %v", input, err)
		}
		if actual != expected {
			t.Fatalf("parseAmount(%q) = %d, want %d", input, actual, expected)
		}
	}
}

func TestParseAmountRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "nol", "0", "-1000"} {
		if _, err := parseAmount(input); err == nil {
			t.Fatalf("parseAmount(%q) should fail", input)
		}
	}
}

func TestParseCommand(t *testing.T) {
	command, args := parseCommand("/keluar@hubby_bot 25rb makan siang")
	if command != "keluar" || args != "25rb makan siang" {
		t.Fatalf("unexpected command=%q args=%q", command, args)
	}
}

func TestFormatRupiah(t *testing.T) {
	if actual := formatRupiah(-1_250_000); actual != "-Rp1.250.000" {
		t.Fatalf("formatRupiah returned %q", actual)
	}
}
