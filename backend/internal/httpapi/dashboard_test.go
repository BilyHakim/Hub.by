package httpapi

import "testing"

func TestPercentageChange(t *testing.T) {
	tests := []struct {
		name              string
		current, previous int64
		want              *float64
	}{
		{name: "increase", current: 125, previous: 100, want: floatPointer(25)},
		{name: "decrease", current: 75, previous: 100, want: floatPointer(-25)},
		{name: "negative baseline", current: 50, previous: -50, want: floatPointer(200)},
		{name: "zero baseline", current: 100, previous: 0, want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := percentageChange(test.current, test.previous)
			if test.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", *got)
				}
				return
			}
			if got == nil || *got != *test.want {
				t.Fatalf("got %v, want %v", got, *test.want)
			}
		})
	}
}

func floatPointer(value float64) *float64 {
	return &value
}
