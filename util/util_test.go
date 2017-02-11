package util

import "testing"

func TestStripElements(t *testing.T) {
	for _, test := range []struct {
		elements   []string
		candidates []string
		expected   []string
	}{
		{[]string{}, []string{}, []string{}},
		{[]string{"foo"}, []string{}, []string{"foo"}},
		{[]string{"foo"}, []string{"foo"}, []string{}},
		{[]string{"foo"}, []string{"bar"}, []string{"foo"}},
		{[]string{"foo", "bar"}, []string{"foo", "bar"}, []string{}},
		{[]string{"foo", "foo", "bar"}, []string{"foo", "bar"}, []string{}},
	} {
		got := StripElements(test.elements, test.candidates...)

		if len(got) != len(test.expected) {
			t.Fatalf("stripElements(%q, %q) => %q, want %q", test.elements, test.candidates, got, test.expected)
		}

		for i := range test.expected {
			if test.expected[i] != got[i] {
				t.Fatalf("stripElements(%q, %q) => %q, want %q", test.elements, test.candidates, got, test.expected)
			}
		}
	}
}
