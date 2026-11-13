package main

import "strings"

// LinjeSjekk er sammenligningen av én utskriftslinje.
type LinjeSjekk struct {
	Nr    int
	Ditt  string // med synlige mellomrom
	Fasit string // med synlige mellomrom
	OK    bool
}

// Resultat er hele sammenligningen av et svar mot fasiten.
type Resultat struct {
	Linjer     []LinjeSjekk
	Riktige    int
	Totalt     int
	AlleRiktig bool
}

// normaliser deler i linjer, godtar \r\n og \r, fjerner blanke på slutten av
// hver linje og tomme linjer helt til slutt. Mellomrom INNI og FORAN en linje
// teller med - det er der %3i og %-5s gjør forskjell.
func normaliser(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	linjer := strings.Split(s, "\n")
	for i := range linjer {
		linjer[i] = strings.TrimRight(linjer[i], " \t")
	}
	for len(linjer) > 0 && linjer[len(linjer)-1] == "" {
		linjer = linjer[:len(linjer)-1]
	}
	return linjer
}

func synlig(s string) string { return strings.ReplaceAll(s, " ", "·") }

// Sjekk sammenligner et svar med fasiten, linje for linje.
func Sjekk(svar, fasit string) Resultat {
	d, f := normaliser(svar), normaliser(fasit)
	var res Resultat
	for i := 0; i < max(len(d), len(f)); i++ {
		var di, fi string
		if i < len(d) {
			di = d[i]
		}
		if i < len(f) {
			fi = f[i]
		}
		ok := i < len(d) && i < len(f) && di == fi
		if ok {
			res.Riktige++
		}
		res.Linjer = append(res.Linjer, LinjeSjekk{Nr: i + 1, Ditt: synlig(di), Fasit: synlig(fi), OK: ok})
	}
	res.Totalt = len(f)
	res.AlleRiktig = len(d) == len(f) && res.Riktige == len(f)
	return res
}
