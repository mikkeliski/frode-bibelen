package main

import (
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestBytNavn(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	tok, _ := tavle.BliMed("Stygt Navn")
	tavle.BliMed("Opptatt")
	tavle.Svar(tok, "0-0000000000000001", true)

	cases := []struct {
		gammelt, nytt string
		feil          error
	}{
		{"finnes ikke", "Hva Som Helst", ErrUkjentNavn},
		{"stygt navn", "opptatt", ErrNavnOpptatt},
		{"Stygt Navn", "<b>x</b>", ErrUgyldigNavn},
		{"Stygt Navn", "x", ErrUgyldigNavn},
		{"stygt   navn", "  Pent   Navn ", nil}, // renses, og store/små ignoreres i det gamle
		{"Pent Navn", "PENT navn", nil},         // samme spiller kan endre store/små bokstaver
	}
	for _, c := range cases {
		if err := tavle.BytNavn(c.gammelt, c.nytt); !errors.Is(err, c.feil) {
			t.Errorf("BytNavn(%q, %q) = %v, ville ha %v", c.gammelt, c.nytt, err, c.feil)
		}
	}
	info := tavle.Info(tok)
	if info == nil || info.Navn != "PENT navn" || info.Poeng != 3 {
		t.Fatalf("etter navnebytte: %+v (cookien og poengene skal beholdes)", info)
	}
	if _, err := tavle.BliMed("Stygt Navn"); err != nil {
		t.Errorf("det gamle navnet skulle blitt ledig: %v", err)
	}
	if _, err := tavle.BliMed("pent navn"); !errors.Is(err, ErrNavnOpptatt) {
		t.Errorf("det nye navnet skulle vært opptatt: %v", err)
	}
}

func TestByttNavnIFil(t *testing.T) {
	fil := filepath.Join(t.TempDir(), "poeng.json")
	tavle, _ := nyPoengtavle(fil, 100)
	tok, _ := tavle.BliMed("Upassende")
	tavle.BliMed("Annen")
	tavle.Lagre()
	logg := slog.New(slog.NewTextHandler(io.Discard, nil))

	for _, c := range [][2]string{{"", "Nytt"}, {"Upassende", ""}, {"Ukjent", "Nytt"}, {"Upassende", "annen"}} {
		if kode := byttNavnIFil(fil, c[0], c[1], logg); kode != 1 {
			t.Errorf("byttNavnIFil(%q, %q) = %d, ville ha 1", c[0], c[1], kode)
		}
	}
	if kode := byttNavnIFil("", "Upassende", "Nytt", logg); kode != 1 {
		t.Error("uten -poengfil skulle det feilet")
	}
	if kode := byttNavnIFil(fil, "upassende", "Grei Gutt", logg); kode != 0 {
		t.Fatalf("kode %d", kode)
	}
	ny, _ := nyPoengtavle(fil, 100)
	if info := ny.Info(tok); info == nil || info.Navn != "Grei Gutt" {
		t.Errorf("etter lasting: %+v", info)
	}
	if ny.Antall() != 2 {
		t.Errorf("%d spillere, ville ha 2", ny.Antall())
	}
}
