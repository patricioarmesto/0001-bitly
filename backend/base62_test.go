package main

import "testing"

func TestEncodeBase62(t *testing.T) {
	cases := []struct {
		id   uint64
		want string
	}{
		{0, "0"},
		{61, "Z"},
		{62, "10"},
		{12345, "3d7"},
		{264905530269528064, "jzgKpHNygg"}, // Snowflake de 2026-01-01, worker 7, secuencia 0
	}

	for _, test := range cases {
		if got := encodeBase62(test.id); got != test.want {
			t.Errorf("encodeBase62(%d) = %q, esperaba %q", test.id, got, test.want)
		}
	}
}

func TestDecodeBase62(t *testing.T) {
	cases := []struct {
		code string
		want uint64
	}{
		{"0", 0},
		{"Z", 61},
		{"10", 62},
		{"3d7", 12345},
		{"jzgKpHNygg", 264905530269528064},
	}

	for _, test := range cases {
		if got := decodeBase62(test.code); got != test.want {
			t.Errorf("decodeBase62(%q) = %d, esperaba %d", test.code, got, test.want)
		}
	}
}
