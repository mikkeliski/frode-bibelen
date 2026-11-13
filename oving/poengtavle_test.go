package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestKallenavn(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	for _, navn := range []string{"Mikkel", "Øystein Å", "ola_99", "K.L-1", "ab", "  Kari   Nordmann  "} {
		if _, err := tavle.BliMed(navn); err != nil {
			t.Errorf("BliMed(%q) = %v, skulle gått", navn, err)
		}
	}
	for _, navn := range []string{"", "a", "<script>", "x\"y",
		strings.Repeat("a", 21), "emoji🙂", "a/b", "a'b", "\xff\xfe"} {
		if _, err := tavle.BliMed(navn); err == nil {
			t.Errorf("BliMed(%q) skulle feilet", navn)
		}
	}
	if _, err := tavle.BliMed("MIKKEL"); err != ErrNavnOpptatt {
		t.Errorf("samme navn med andre store/små bokstaver: %v", err)
	}
	if _, err := tavle.BliMed("Kari Nordmann"); err != ErrNavnOpptatt {
		t.Errorf("navnet skulle vært renset og opptatt: %v", err)
	}
	// Linjeskift og tabulatorer blir til ett mellomrom - aldri med i navnet.
	tok, err := tavle.BliMed("Tab\tog\nLinje")
	if err != nil || tavle.Info(tok).Navn != "Tab og Linje" {
		t.Errorf("blanke ble ikke renset: %v %+v", err, tavle.Info(tok))
	}
}

func TestPoengregler(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	tok, _ := tavle.BliMed("Spiller")
	sjekk := func(id string, riktig bool, vil int) {
		t.Helper()
		if p, _ := tavle.Svar(tok, id, riktig); p != vil {
			t.Errorf("%s riktig=%v: %d poeng, ville ha %d", id, riktig, p, vil)
		}
	}
	sjekk("0-0000000000000001", true, 3) // første forsøk
	sjekk("0-0000000000000001", true, 0) // allerede løst
	sjekk("0-0000000000000002", false, 0)
	sjekk("0-0000000000000002", true, 2) // andre forsøk
	sjekk("0-0000000000000003", false, 0)
	sjekk("0-0000000000000003", false, 0)
	sjekk("0-0000000000000003", false, 0)
	sjekk("0-0000000000000003", true, 1) // minst 1
	tavle.FasitVist(tok, "0-0000000000000004")
	sjekk("0-0000000000000004", true, 0) // fasit vist

	info := tavle.Info(tok)
	if info.Poeng != 6 || info.Loste != 3 || info.Plass != 1 {
		t.Errorf("info = %+v", info)
	}
	if !tavle.Ferdig(tok, "0-0000000000000004") || tavle.Ferdig(tok, "0-0000000000000005") {
		t.Error("Ferdig gir feil svar")
	}
	if rad := tavle.Topp(10, tok)[0]; rad.Prosent != 33 || !rad.Meg {
		t.Errorf("rad = %+v", rad)
	}
}

func TestUkjenteTokenGirIngenting(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	tavle.BliMed("Noen")
	for _, tok := range []string{"", "abc", strings.Repeat("!", 43), strings.Repeat("A", 43), strings.Repeat("A", 400)} {
		if tavle.Info(tok) != nil || tavle.Slett(tok) {
			t.Errorf("token %q ga tilgang", tok)
		}
		if p, m := tavle.Svar(tok, "0-0000000000000001", true); p != 0 || m != "" {
			t.Errorf("token %q fikk poeng", tok)
		}
	}
}

func TestSorteringOgTopp(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	var tok []string
	for _, n := range []string{"Aa", "Bb", "Cc", "Dd"} {
		x, _ := tavle.BliMed(n)
		tok = append(tok, x)
	}
	tavle.Svar(tok[2], "0-0000000000000001", true) // Cc: 3
	tavle.Svar(tok[1], "0-0000000000000001", false)
	tavle.Svar(tok[1], "0-0000000000000001", true) // Bb: 2
	rader := tavle.Topp(2, tok[3])
	if len(rader) != 3 || rader[0].Navn != "Cc" || rader[1].Navn != "Bb" || !rader[2].Meg || rader[2].Navn != "Dd" {
		t.Errorf("rader = %+v", rader)
	}
}

func TestTavlaFull(t *testing.T) {
	tavle, _ := nyPoengtavle("", 2)
	tavle.BliMed("En1")
	tavle.BliMed("To2")
	if _, err := tavle.BliMed("Tre3"); err != ErrTavlaFull {
		t.Errorf("err = %v, ville ha ErrTavlaFull", err)
	}
}

func TestLagreOgLaste(t *testing.T) {
	fil := filepath.Join(t.TempDir(), "poeng.json")
	tavle, _ := nyPoengtavle(fil, 100)
	tok, _ := tavle.BliMed("Lagret Spiller")
	tavle.Svar(tok, "1-0000000000000001", true)
	if err := tavle.Lagre(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(fil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("rettigheter %o, ville ha 600", st.Mode().Perm())
	}
	data, _ := os.ReadFile(fil)
	if strings.Contains(string(data), tok) {
		t.Error("tokenet ligger i klartekst i fila")
	}
	ny, err := nyPoengtavle(fil, 100)
	if err != nil {
		t.Fatal(err)
	}
	if info := ny.Info(tok); info == nil || info.Poeng != 3 || info.Navn != "Lagret Spiller" {
		t.Errorf("etter lasting: %+v", info)
	}
	if !ny.Ferdig(tok, "1-0000000000000001") {
		t.Error("løste oppgaver ble ikke husket")
	}
	if _, err := ny.BliMed("lagret spiller"); err != ErrNavnOpptatt {
		t.Error("navneregisteret ble ikke lastet")
	}
}

func TestSamtidigeSvar(t *testing.T) {
	tavle, _ := nyPoengtavle(filepath.Join(t.TempDir(), "p.json"), 100)
	tok, _ := tavle.BliMed("Rask")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tavle.Svar(tok, LagID(0, uint64(i)), true)
			tavle.Topp(10, tok)
			_ = tavle.Lagre()
		}(i)
	}
	wg.Wait()
	if info := tavle.Info(tok); info.Poeng != 150 {
		t.Errorf("poeng = %d, ville ha 150", info.Poeng)
	}
}

// ---------------------------------------------------------- via HTTP ---

func klientMedCookies(t *testing.T) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// les(t)(k.Get(...)) leser hele svaret som tekst.
func les(t *testing.T) func(*http.Response, error) string {
	return func(res *http.Response, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return string(b)
	}
}

func TestPoengtavleViaHTTP(t *testing.T) {
	s := testServer(t, konfig{})
	k := klientMedCookies(t)

	// Bli med
	res, err := k.PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {"Testesen"}})
	side := les(t)(res, err)
	if !strings.Contains(side, "Du spiller som <strong>Testesen</strong>") {
		t.Fatal("ble ikke med på tavla")
	}
	var cookie *http.Cookie
	for _, c := range res.Request.Response.Cookies() {
		if c.Name == cookieNavn {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("cookie mangler eller har feil flagg: %+v", cookie)
	}

	// Riktig svar gir 3 poeng, og samme oppgave gir ikke flere
	o, _ := HentOppgave(LagID(0, 99))
	side = les(t)(k.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {o.Fasit}, "handling": {"sjekk"}}))
	if !strings.Contains(side, "3 poeng – riktig på første forsøk") {
		t.Error("fikk ikke 3 poeng for riktig svar")
	}
	side = les(t)(k.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {o.Fasit}, "handling": {"sjekk"}}))
	if strings.Contains(side, "3 poeng – riktig") || !strings.Contains(side, "Ingen nye poeng") {
		t.Error("fikk poeng to ganger for samme oppgave")
	}

	// Tomt svar er ikke et forsøk
	o2, _ := HentOppgave(LagID(1, 99))
	les(t)(k.PostForm(s.URL+"/oppgave/"+o2.ID, url.Values{"svar": {"  "}, "handling": {"sjekk"}}))
	side = les(t)(k.PostForm(s.URL+"/oppgave/"+o2.ID, url.Values{"svar": {o2.Fasit}, "handling": {"sjekk"}}))
	if !strings.Contains(side, "3 poeng – riktig på første forsøk") {
		t.Error("tomt svar ble telt som galt forsøk")
	}

	// Samme kallenavn fra en annen klient avvises
	res, err = klientMedCookies(t).PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {"testesen"}})
	if les(t)(res, err); res.StatusCode != http.StatusConflict {
		t.Errorf("duplikat: status %d, ville ha 409", res.StatusCode)
	}

	// Tavla viser poengene
	if side = les(t)(k.Get(s.URL + "/poeng")); !strings.Contains(side, "<strong>6</strong>") {
		t.Error("tavla viser ikke 6 poeng")
	}

	// Sletting uten bekreftelse gjør ingenting, med bekreftelse fjernes spilleren
	les(t)(k.PostForm(s.URL+"/poeng/slett", url.Values{}))
	if !strings.Contains(les(t)(k.Get(s.URL+"/poeng")), "Testesen") {
		t.Error("slettet uten bekreftelse")
	}
	side = les(t)(k.PostForm(s.URL+"/poeng/slett", url.Values{"bekreft": {"ja"}}))
	if strings.Contains(side, "Testesen") || !strings.Contains(side, "Bli med") {
		t.Error("spilleren ble ikke slettet")
	}
}

func TestUgyldigKallenavnEscapes(t *testing.T) {
	s := testServer(t, konfig{})
	res, err := http.PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {`<img src=x onerror=alert(1)>`}})
	side := les(t)(res, err)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, ville ha 400", res.StatusCode)
	}
	if strings.Contains(side, "<img src=x") {
		t.Error("kallenavnet ble ikke escapet")
	}
}

func TestRegistreringBegrenses(t *testing.T) {
	s := testServer(t, konfig{})
	var sist int
	for i := 0; i < registreringTopp+1; i++ {
		res, err := http.PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {"Spiller" + string(rune('A'+i))}})
		les(t)(res, err)
		sist = res.StatusCode
	}
	if sist != http.StatusTooManyRequests {
		t.Errorf("status %d etter %d registreringer, ville ha 429", sist, registreringTopp+1)
	}
}

func TestSlettFraFil(t *testing.T) {
	fil := filepath.Join(t.TempDir(), "poeng.json")
	tavle, _ := nyPoengtavle(fil, 100)
	tavle.BliMed("Upassende")
	tavle.BliMed("Grei")
	tavle.Lagre()
	logg := slog.New(slog.NewTextHandler(io.Discard, nil))
	if kode := slettFraFil(fil, "upassende", logg); kode != 0 {
		t.Fatalf("kode %d", kode)
	}
	ny, _ := nyPoengtavle(fil, 100)
	if ny.Antall() != 1 {
		t.Errorf("%d spillere igjen, ville ha 1", ny.Antall())
	}
}

// ------------------------------------------------- juks med fasiten ---

func TestVistFasitGirIngenPoeng(t *testing.T) {
	tavle, _ := nyPoengtavle("", 100)
	a, _ := tavle.BliMed("Anne")
	b, _ := tavle.BliMed("Berit")
	id := "2-0000000000000042"
	tavle.Svar(a, id, true) // Anne løser først: 3 poeng
	tavle.Avslor(id)        // fasiten vises (for Anne, eller hvem som helst)
	if p, _ := tavle.Svar(b, id, true); p != 0 {
		t.Errorf("Berit fikk %d poeng for en oppgave med vist fasit", p)
	}
	if !tavle.Ferdig(b, id) {
		t.Error("oppgaven skulle vært merket som ferdig for Berit")
	}
	if tavle.Info(a).Poeng != 3 {
		t.Error("Anne skulle beholdt poengene sine")
	}
}

func harFasit(side string) bool {
	return strings.Contains(side, `class="ut"`) || strings.Contains(side, "<th>Fasit</th>")
}

func TestGaltSvarViserIkkeFasit(t *testing.T) {
	s := testServer(t, konfig{})
	o, _ := HentOppgave(LagID(3, 7))
	anonym := http.DefaultClient
	innlogget := klientMedCookies(t)
	les(t)(innlogget.PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {"Prover"}}))

	for navn, k := range map[string]*http.Client{"anonym": anonym, "innlogget": innlogget} {
		side := les(t)(k.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {"tull"}, "handling": {"sjekk"}}))
		if harFasit(side) {
			t.Errorf("%s: galt svar viste fasiten", navn)
		}
		if !strings.Contains(side, "✗") {
			t.Errorf("%s: galt svar viste ikke hvilke linjer som er feil", navn)
		}
	}
	// Galt svar ga ingen fasit, så riktig svar nå gir fortsatt 2 poeng.
	side := les(t)(innlogget.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {o.Fasit}, "handling": {"sjekk"}}))
	if !harFasit(side) || !strings.Contains(side, "2 poeng") {
		t.Error("riktig svar etter galt skulle gitt 2 poeng og vist fasiten")
	}
}

func TestFasitIPrivatVinduGirIngenPoeng(t *testing.T) {
	s := testServer(t, konfig{})
	innlogget := klientMedCookies(t)
	les(t)(innlogget.PostForm(s.URL+"/poeng/bli-med", url.Values{"navn": {"Lur"}}))

	// 1) «Vis fasit» uten cookie, så riktig svar innlogget
	o, _ := HentOppgave(LagID(4, 1))
	if side := les(t)(http.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"handling": {"fasit"}})); !harFasit(side) {
		t.Fatal("anonym «Vis fasit» viste ikke fasiten")
	}
	side := les(t)(innlogget.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {o.Fasit}, "handling": {"sjekk"}}))
	if !strings.Contains(side, "Ingen poeng: fasiten til denne oppgaven er allerede vist") {
		t.Error("fikk poeng etter at fasiten ble vist i et annet vindu")
	}

	// 2) Riktig svar uten cookie (f.eks. fra en venn), så riktig svar innlogget
	o2, _ := HentOppgave(LagID(5, 1))
	les(t)(http.PostForm(s.URL+"/oppgave/"+o2.ID, url.Values{"svar": {o2.Fasit}, "handling": {"sjekk"}}))
	side = les(t)(innlogget.PostForm(s.URL+"/oppgave/"+o2.ID, url.Values{"svar": {o2.Fasit}, "handling": {"sjekk"}}))
	if strings.Contains(side, "riktig på første forsøk") {
		t.Error("fikk poeng for en oppgave der fasiten var vist for en annen")
	}

	if side = les(t)(innlogget.Get(s.URL + "/poeng")); !strings.Contains(side, "· 0 poeng ·") {
		t.Error("spilleren skulle hatt 0 poeng")
	}
	// Oppgavesiden sier fra om at oppgaven ikke gir poeng
	if side = les(t)(innlogget.Get(s.URL + "/oppgave/" + o.ID)); !strings.Contains(side, "denne oppgaven gir ikke flere poeng") {
		t.Error("oppgavesiden sa ikke fra om at fasiten er vist")
	}
}

func TestAvslorteLagresOgRyddes(t *testing.T) {
	fil := filepath.Join(t.TempDir(), "poeng.json")
	tavle, _ := nyPoengtavle(fil, 100)
	tok, _ := tavle.BliMed("Huskes")
	tavle.Avslor("6-0000000000000001")
	if err := tavle.Lagre(); err != nil {
		t.Fatal(err)
	}
	ny, _ := nyPoengtavle(fil, 100)
	if p, _ := ny.Svar(tok, "6-0000000000000001", true); p != 0 {
		t.Error("viste fasiter ble ikke husket etter omstart")
	}

	// Når tavla er full, ryddes de eldste - den holder seg under taket.
	start := time.Unix(1_700_000_000, 0)
	n := 0
	tavle.na = func() time.Time { n++; return start.Add(time.Duration(n) * time.Second) }
	for i := 0; i < maksAvslorte+10; i++ {
		tavle.Avslor(LagID(7, uint64(i)))
	}
	if len(tavle.avslort) > maksAvslorte {
		t.Errorf("%d avslørte, taket er %d", len(tavle.avslort), maksAvslorte)
	}
	if _, nyest := tavle.avslort[LagID(7, uint64(maksAvslorte+9))]; !nyest {
		t.Error("den nyeste ble ryddet bort")
	}
}
