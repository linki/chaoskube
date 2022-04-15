package main

import (
	"testing"
)

func TestIsOneNamespace(t *testing.T) {
	testCases := []struct {
		strIn string
		want  bool
	}{
		{
			strIn: "test",
			want:  true,
		},
		{
			strIn: "",
			want:  false,
		},
		{
			strIn: "test,default",
			want:  false,
		},
		{
			strIn: "!test",
			want:  false,
		},
		{
			strIn: "test,!default",
			want:  false,
		},
	}

	for _, tc := range testCases {
		rezult := isOneNamespace(tc.strIn)
		if rezult != tc.want {
			t.Errorf("isOneNamespace(%s) want %t, got %t", tc.strIn, tc.want, rezult)
		}
	}
}
