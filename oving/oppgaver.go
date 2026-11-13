package main

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
)

// Oppgave er én generert «Hva blir utskriften?»-oppgave.
type Oppgave struct {
	ID     string   // Stabil ID: samme ID gir alltid samme oppgave.
	Del    string   // "A" (tekst og pekere) eller "B" (struct og funksjoner).
	Tittel string   // Kort beskrivelse av malen.
	Kode   string   // Komplett C-program.
	Fasit  string   // Nøyaktig det programmet skriver ut.
	Tips   []string // Vises etter at svaret er sjekket.
}

// Linjer returnerer antall linjer i utskriften (som Frode oppgir på eksamen).
func (o Oppgave) Linjer() int {
	return len(strings.Split(strings.TrimSuffix(o.Fasit, "\n"), "\n"))
}

// KodeLinjer deler koden i linjer for visning med linjenummer.
func (o Oppgave) KodeLinjer() []string {
	return strings.Split(strings.TrimSuffix(o.Kode, "\n"), "\n")
}

type mal struct {
	del    string
	tittel string
	lag    func(r *rand.Rand) (kode, fasit string, tips []string)
}

// maler: rekkefølgen er en del av ID-formatet - legg bare til nye på slutten.
var maler = []mal{
	{"A", "Indekser og hopp i en tekst", malIndekshopp},
	{"A", "Pekere og *t++", malPekere},
	{"A", "strchr og strrchr", malStrchr},
	{"A", "strncmp, strstr og strcmp", malSammenlign},
	{"A", "Tegnaritmetikk og toupper/tolower", malTegnaritmetikk},
	{"A", "Bygge en ny tekst", malByggTekst},
	{"B", "Struct som kopi eller peker", malStructKopi},
	{"B", "Bytte verdier og referanseoverføring", malBytt},
	{"B", "Array av structer og funksjoner", malVarer},
	{"B", "Funksjon i løkke og heltallsdivisjon", malTallrekke},
}

var idMonster = regexp.MustCompile(`^([0-9]{1,2})-([0-9a-f]{16})$`)

// ErrUgyldigID betyr at ID-en ikke har riktig format eller peker på en ukjent mal.
var ErrUgyldigID = errors.New("ugyldig oppgave-ID")

// LagID lager en ID for mal nr. malNr med gitt frø.
func LagID(malNr int, fro uint64) string {
	return fmt.Sprintf("%d-%016x", malNr, fro)
}

// HentOppgave lager oppgaven som hører til en ID. Samme ID gir alltid samme oppgave.
func HentOppgave(id string) (Oppgave, error) {
	m := idMonster.FindStringSubmatch(id)
	if m == nil {
		return Oppgave{}, ErrUgyldigID
	}
	malNr, err := strconv.Atoi(m[1])
	if err != nil || malNr < 0 || malNr >= len(maler) {
		return Oppgave{}, ErrUgyldigID
	}
	fro, err := strconv.ParseUint(m[2], 16, 64)
	if err != nil {
		return Oppgave{}, ErrUgyldigID
	}
	r := rand.New(rand.NewPCG(fro, 0x9e3779b97f4a7c15))
	kode, fasit, tips := maler[malNr].lag(r)
	return Oppgave{
		ID:     LagID(malNr, fro),
		Del:    maler[malNr].del,
		Tittel: maler[malNr].tittel,
		Kode:   kode,
		Fasit:  fasit,
		Tips:   tips,
	}, nil
}

// NyID trekker en tilfeldig mal (fra del "A", "B" eller alle ved "") og et tilfeldig frø.
func NyID(del string) string {
	var kandidater []int
	for i, m := range maler {
		if del == "" || m.del == del {
			kandidater = append(kandidater, i)
		}
	}
	if len(kandidater) == 0 {
		for i := range maler {
			kandidater = append(kandidater, i)
		}
	}
	return LagID(kandidater[rand.IntN(len(kandidater))], rand.Uint64())
}

// ------------------------------------------------------------- hjelpere ---

var byer = []string{
	"Gjovik", "Hamar", "Lillehammer", "Trondheim", "Bergen", "Tromso", "Bodo",
	"Molde", "Alesund", "Narvik", "Halden", "Larvik", "Skien", "Arendal",
	"Kongsberg", "Drammen", "Elverum", "Steinkjer", "Levanger", "Namsos",
	"Harstad", "Kirkenes", "Raufoss", "Brumunddal", "Mosjoen", "Notodden",
	"Fredrikstad", "Honefoss", "Kristiansund", "Orkanger",
}

var fornavn = []string{
	"Mikkel", "Oskar", "Johan", "Snorre", "Frode", "Aslak", "Kelly", "Richard", "Erjon",
}

var varer = []string{"Monster", "Redbull", "Polse", "Pringles", "Kaffe", "Fjordlands", "Powerking", "IsKaffe"}

func velg[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

// mellom gir et tilfeldig heltall i [a, b].
func mellom(r *rand.Rand, a, b int) int { return a + r.IntN(b-a+1) }

// lagTekst lager en tekst av 2-3 ulike bynavn, f.eks. "Gjovik-Hamar".
func lagTekst(r *rand.Rand) string {
	n := mellom(r, 2, 3)
	sep := velg(r, []string{"-", " ", "-"})
	brukt := map[string]bool{}
	var deler []string
	for len(deler) < n {
		b := velg(r, byer)
		if !brukt[b] && len(strings.Join(append(deler, b), sep)) <= 32 {
			brukt[b] = true
			deler = append(deler, b)
		} else if len(deler) >= 2 {
			break
		}
	}
	return strings.Join(deler, sep)
}

func erBokstav(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func tilStor(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 'a' + 'A'
	}
	return c
}

func tilLiten(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

// endre lager en variant av s som garantert er ulik (bytter store/små eller ett tegn).
func endre(r *rand.Rand, s string) string {
	b := []byte(s)
	i := r.IntN(len(b))
	switch {
	case b[i] >= 'a' && b[i] <= 'z':
		b[i] = tilStor(b[i])
	case b[i] >= 'A' && b[i] <= 'Z':
		b[i] = tilLiten(b[i])
	default:
		b[i] = 'x'
	}
	return string(b)
}

// ikkeIBruk finner en liten bokstav som ikke finnes i s (eller 'q' som nødløsning).
func ikkeIBruk(r *rand.Rand, s string) byte {
	var ledige []byte
	for c := byte('a'); c <= 'z'; c++ {
		if !strings.ContainsRune(s, rune(c)) {
			ledige = append(ledige, c)
		}
	}
	if len(ledige) == 0 {
		return 'q'
	}
	return velg(r, ledige)
}

// bokstavIndekser gir indeksene til alle bokstaver i s.
func bokstavIndekser(s string) []int {
	var ix []int
	for i := 0; i < len(s); i++ {
		if erBokstav(s[i]) {
			ix = append(ix, i)
		}
	}
	return ix
}

// tegnLiteral skriver et tegn som C-tegnkonstant.
func tegnLiteral(c byte) string { return "'" + string(c) + "'" }
