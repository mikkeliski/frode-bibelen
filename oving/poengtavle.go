package main

// Poengtavle: frivillig. Bare de som velger et kallenavn får en cookie.
// Tokenet i cookien lagres aldri i klartekst - bare som SHA-256 - så den
// lagrede fila kan ikke brukes til å utgi seg for noen.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maksNavnTegn      = 20
	minNavnTegn       = 2
	maksFerdige       = 50_000 // oppgaver per spiller som huskes som løst/fasit vist
	maksApneForsok    = 1_000  // uløste oppgaver med gale forsøk per spiller
	tokenBytes        = 32
	maksAvslorte      = 100_000             // oppgaver der fasiten er vist, som huskes
	avslortVarighet   = 60 * 24 * time.Hour // så lenge en vist fasit huskes
	poengtavleVersjon = 1
)

var (
	ErrUgyldigNavn = errors.New("ugyldig kallenavn")
	ErrNavnOpptatt = errors.New("kallenavnet er opptatt")
	ErrTavlaFull   = errors.New("poengtavla er full")
	ErrUkjentNavn  = errors.New("fant ingen spiller med det kallenavnet")
)

type spiller struct {
	TokenHash string          `json:"token_sha256"`
	Navn      string          `json:"navn"`
	Poeng     int             `json:"poeng"`
	Loste     int             `json:"loste"`
	Forste    int             `json:"forste_forsok"`
	Opprettet time.Time       `json:"opprettet"`
	SistAktiv time.Time       `json:"sist_aktiv"`
	Ferdige   map[string]bool `json:"ferdige"` // oppgave-ID -> løst eller fasit vist
	Forsok    map[string]int  `json:"forsok"`  // oppgave-ID -> antall gale svar (uløste)
}

// Rad er én linje på poengtavla (kopi - trygg å bruke i maler).
type Rad struct {
	Plass   int
	Navn    string
	Poeng   int
	Loste   int
	Prosent int // andel løst på første forsøk
	Meg     bool
}

// SpillerInfo er det siden viser om den innloggede spilleren.
type SpillerInfo struct {
	Navn  string
	Poeng int
	Loste int
	Plass int
}

type poengtavle struct {
	mu       sync.Mutex
	spillere map[string]*spiller // token-hash -> spiller
	navn     map[string]string   // kallenavn i små bokstaver -> token-hash
	avslort  map[string]int64    // oppgave-ID -> når fasiten ble vist (Unix-tid), for ALLE besøkende
	fil      string              // "" = bare i minnet
	maks     int
	endret   bool
	na       func() time.Time
}

type lagretTavle struct {
	Versjon  int              `json:"versjon"`
	Spillere []*spiller       `json:"spillere"`
	Avslorte map[string]int64 `json:"avslorte_fasiter,omitempty"`
}

// nyPoengtavle laster fila hvis den finnes. fil = "" gir en tavle bare i minnet.
func nyPoengtavle(fil string, maks int) (*poengtavle, error) {
	t := &poengtavle{
		spillere: map[string]*spiller{},
		navn:     map[string]string{},
		avslort:  map[string]int64{},
		fil:      fil,
		maks:     maks,
		na:       time.Now,
	}
	if fil == "" {
		return t, nil
	}
	data, err := os.ReadFile(fil)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	var l lagretTavle
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	for _, s := range l.Spillere {
		if s == nil || len(s.TokenHash) != 64 || gyldigNavn(s.Navn) != nil {
			continue
		}
		if s.Ferdige == nil {
			s.Ferdige = map[string]bool{}
		}
		if s.Forsok == nil {
			s.Forsok = map[string]int{}
		}
		t.spillere[s.TokenHash] = s
		t.navn[navneNokkel(s.Navn)] = s.TokenHash
	}
	for id, tid := range l.Avslorte {
		if idMonster.MatchString(id) && len(t.avslort) < maksAvslorte {
			t.avslort[id] = tid
		}
	}
	return t, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// gyldigToken sjekker formatet før vi i det hele tatt hasher noe.
func gyldigToken(token string) bool {
	if len(token) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil
}

func navneNokkel(navn string) string { return strings.ToLower(navn) }

// rensNavn fjerner blanke i endene og slår sammen mellomrom.
func rensNavn(navn string) string { return strings.Join(strings.Fields(navn), " ") }

// gyldigNavn: 2-20 tegn, bokstaver (også æøå), sifre, mellomrom og - _ .
func gyldigNavn(navn string) error {
	if !utf8.ValidString(navn) {
		return ErrUgyldigNavn
	}
	n := utf8.RuneCountInString(navn)
	if n < minNavnTegn || n > maksNavnTegn || navn != rensNavn(navn) {
		return ErrUgyldigNavn
	}
	for _, r := range navn {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" -_.", r) {
			return ErrUgyldigNavn
		}
	}
	return nil
}

// BliMed lager en ny spiller og returnerer tokenet som skal i cookien.
func (t *poengtavle) BliMed(navn string) (string, error) {
	navn = rensNavn(navn)
	if err := gyldigNavn(navn); err != nil {
		return "", err
	}
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, finnes := t.navn[navneNokkel(navn)]; finnes {
		return "", ErrNavnOpptatt
	}
	if len(t.spillere) >= t.maks {
		return "", ErrTavlaFull
	}
	na := t.na()
	h := hashToken(token)
	t.spillere[h] = &spiller{
		TokenHash: h, Navn: navn, Opprettet: na, SistAktiv: na,
		Ferdige: map[string]bool{}, Forsok: map[string]int{},
	}
	t.navn[navneNokkel(navn)] = h
	t.endret = true
	return token, nil
}

// finnLaast krever t.mu.
func (t *poengtavle) finnLaast(token string) *spiller {
	if !gyldigToken(token) {
		return nil
	}
	return t.spillere[hashToken(token)]
}

// Info gir spillerens navn, poeng og plass, eller nil hvis tokenet er ukjent.
func (t *poengtavle) Info(token string) *SpillerInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.finnLaast(token)
	if s == nil {
		return nil
	}
	return &SpillerInfo{Navn: s.Navn, Poeng: s.Poeng, Loste: s.Loste, Plass: t.plassLaast(s)}
}

// Ferdig sier om oppgaven ikke lenger kan gi spilleren poeng: den er løst,
// eller fasiten er vist (av spilleren selv eller noen andre).
func (t *poengtavle) Ferdig(token, oppgaveID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.finnLaast(token)
	if s == nil {
		return false
	}
	_, vist := t.avslort[oppgaveID]
	return s.Ferdige[oppgaveID] || vist
}

// Avslor husker at fasiten til en oppgave er vist til noen. Da gir den
// ikke poeng til noen som ikke allerede har løst den - så det hjelper ikke
// å hente fasiten i et privat vindu, en annen nettleser eller en annen konto.
func (t *poengtavle) Avslor(oppgaveID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, finnes := t.avslort[oppgaveID]; finnes {
		return
	}
	if len(t.avslort) >= maksAvslorte {
		t.ryddAvslorteLaast()
	}
	t.avslort[oppgaveID] = t.na().Unix()
	t.endret = true
}

// ryddAvslorteLaast fjerner gamle oppføringer, og om nødvendig den eldste tidelen.
func (t *poengtavle) ryddAvslorteLaast() {
	grense := t.na().Add(-avslortVarighet).Unix()
	for id, tid := range t.avslort {
		if tid < grense {
			delete(t.avslort, id)
		}
	}
	if len(t.avslort) < maksAvslorte {
		return
	}
	tider := make([]int64, 0, len(t.avslort))
	for _, tid := range t.avslort {
		tider = append(tider, tid)
	}
	sort.Slice(tider, func(i, j int) bool { return tider[i] < tider[j] })
	kutt := tider[len(tider)/10]
	for id, tid := range t.avslort {
		if tid <= kutt {
			delete(t.avslort, id)
		}
	}
}

// Svar registrerer et svar og returnerer poengene det ga og en melding.
func (t *poengtavle) Svar(token, oppgaveID string, riktig bool) (int, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.finnLaast(token)
	if s == nil {
		return 0, ""
	}
	s.SistAktiv = t.na()
	t.endret = true
	if s.Ferdige[oppgaveID] {
		return 0, "Ingen nye poeng: du har allerede løst denne oppgaven eller sett fasiten."
	}
	if _, vist := t.avslort[oppgaveID]; vist {
		return 0, "Ingen poeng: fasiten til denne oppgaven er allerede vist – her, i et annet vindu eller for noen andre."
	}
	gale := s.Forsok[oppgaveID]
	if !riktig {
		if len(s.Forsok) >= maksApneForsok {
			clear(s.Forsok)
		}
		s.Forsok[oppgaveID] = gale + 1
		return 0, "Neste riktige svar på denne oppgaven gir " + poengTekst(poengFor(gale+1)) + "."
	}
	p := poengFor(gale)
	s.Poeng += p
	s.Loste++
	if gale == 0 {
		s.Forste++
	}
	delete(s.Forsok, oppgaveID)
	t.merkFerdigLaast(s, oppgaveID)
	if gale == 0 {
		return p, "+" + poengTekst(p) + " – riktig på første forsøk!"
	}
	return p, "+" + poengTekst(p) + "."
}

// FasitVist gjør at oppgaven ikke lenger gir poeng for denne spilleren.
func (t *poengtavle) FasitVist(token, oppgaveID string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.finnLaast(token)
	if s == nil || s.Ferdige[oppgaveID] {
		return ""
	}
	s.SistAktiv = t.na()
	delete(s.Forsok, oppgaveID)
	t.merkFerdigLaast(s, oppgaveID)
	t.endret = true
	return "Fasiten er vist, så denne oppgaven gir ikke poeng lenger."
}

func (t *poengtavle) merkFerdigLaast(s *spiller, oppgaveID string) {
	if len(s.Ferdige) < maksFerdige {
		s.Ferdige[oppgaveID] = true
	}
}

// poengFor: 3 poeng på første forsøk, 2 på andre, deretter 1.
func poengFor(galeForsok int) int { return max(1, 3-galeForsok) }

func poengTekst(p int) string {
	if p == 1 {
		return "1 poeng"
	}
	return string(rune('0'+p)) + " poeng"
}

// Slett fjerner spilleren helt.
func (t *poengtavle) Slett(token string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.finnLaast(token)
	if s == nil {
		return false
	}
	delete(t.navn, navneNokkel(s.Navn))
	delete(t.spillere, s.TokenHash)
	t.endret = true
	return true
}

// BytNavn gir spilleren med kallenavnet gammelt et nytt kallenavn. Poengene og
// cookien beholdes, så spilleren merker bare at navnet er endret.
func (t *poengtavle) BytNavn(gammelt, nytt string) error {
	nytt = rensNavn(nytt)
	if err := gyldigNavn(nytt); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	h, finnes := t.navn[navneNokkel(rensNavn(gammelt))]
	if !finnes {
		return ErrUkjentNavn
	}
	// Samme spiller kan endre store/små bokstaver i sitt eget navn.
	if annen, opptatt := t.navn[navneNokkel(nytt)]; opptatt && annen != h {
		return ErrNavnOpptatt
	}
	s := t.spillere[h]
	delete(t.navn, navneNokkel(s.Navn))
	s.Navn = nytt
	t.navn[navneNokkel(nytt)] = h
	t.endret = true
	return nil
}

// sortertLaast: flest poeng, så flest løste, så den som ble med først.
func (t *poengtavle) sortertLaast() []*spiller {
	alle := make([]*spiller, 0, len(t.spillere))
	for _, s := range t.spillere {
		alle = append(alle, s)
	}
	sort.Slice(alle, func(i, j int) bool {
		a, b := alle[i], alle[j]
		if a.Poeng != b.Poeng {
			return a.Poeng > b.Poeng
		}
		if a.Loste != b.Loste {
			return a.Loste > b.Loste
		}
		if !a.Opprettet.Equal(b.Opprettet) {
			return a.Opprettet.Before(b.Opprettet)
		}
		return a.Navn < b.Navn
	})
	return alle
}

func (t *poengtavle) plassLaast(s *spiller) int {
	for i, x := range t.sortertLaast() {
		if x == s {
			return i + 1
		}
	}
	return 0
}

// Topp gir de n beste. Er spilleren med tokenet utenfor, legges den til sist.
func (t *poengtavle) Topp(n int, token string) []Rad {
	t.mu.Lock()
	defer t.mu.Unlock()
	meg := t.finnLaast(token)
	var rader []Rad
	for i, s := range t.sortertLaast() {
		if i < n || s == meg {
			pros := 0
			if s.Loste > 0 {
				pros = 100 * s.Forste / s.Loste
			}
			rader = append(rader, Rad{Plass: i + 1, Navn: s.Navn, Poeng: s.Poeng,
				Loste: s.Loste, Prosent: pros, Meg: s == meg})
		}
	}
	return rader
}

// Antall gir antall spillere.
func (t *poengtavle) Antall() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.spillere)
}

// Lagre skriver tavla atomisk (midlertidig fil + rename) hvis noe er endret.
func (t *poengtavle) Lagre() error {
	t.mu.Lock()
	if t.fil == "" || !t.endret {
		t.mu.Unlock()
		return nil
	}
	l := lagretTavle{Versjon: poengtavleVersjon, Avslorte: t.avslort}
	for _, s := range t.sortertLaast() {
		l.Spillere = append(l.Spillere, s)
	}
	data, err := json.Marshal(l)
	t.endret = false
	t.mu.Unlock()
	if err == nil {
		err = skrivAtomisk(t.fil, data)
	}
	if err != nil {
		t.mu.Lock()
		t.endret = true
		t.mu.Unlock()
	}
	return err
}

func skrivAtomisk(fil string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(fil), ".poeng-*.tmp") // lages med rettighet 0600
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), fil)
}
