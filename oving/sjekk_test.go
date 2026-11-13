package main

import "testing"

func TestSjekk(t *testing.T) {
	fasit := "  3 10 5\nHamar \nabc\n"
	cases := []struct {
		svar    string
		riktige int
		alle    bool
	}{
		{"  3 10 5\nHamar\nabc", 3, true},
		{"  3 10 5\r\nHamar\r\nabc\r\n\r\n", 3, true},
		{"3 10 5\nHamar\nabc", 2, false}, // mellomrom foran teller
		{"  3 10 5\nHamar", 2, false},
		{"  3 10 5\nHamar\nabc\nekstra", 3, false},
		{"", 0, false},
	}
	for _, c := range cases {
		r := Sjekk(c.svar, fasit)
		if r.Riktige != c.riktige || r.AlleRiktig != c.alle || r.Totalt != 3 {
			t.Errorf("Sjekk(%q) = %d riktige, alle=%v; ville ha %d, %v", c.svar, r.Riktige, r.AlleRiktig, c.riktige, c.alle)
		}
	}
}
