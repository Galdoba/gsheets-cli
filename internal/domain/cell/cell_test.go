package cell

import "testing"

func Test_colToLetter(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		n    int
		want string
	}{
		{name: "simple", n: 3, want: "C"},
		{name: "last letter", n: 26, want: "Z"},
		{name: "double", n: 27, want: "AA"},
		{name: "double 2", n: 28, want: "AB"},
		{name: "zero", n: 0, want: ""},
		{name: "negative", n: -5, want: ""},
		{name: "HUGE", n: 100500, want: "ERQJ"},
		{name: "HUGE", n: 100501, want: "ERQK"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colToLetter(tt.n)
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("colToLetter() = %v, want %v", got, tt.want)
			}
		})
	}
}
