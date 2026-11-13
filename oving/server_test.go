package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testServer(t *testing.T, konf konfig) *httptest.Server {
	t.Helper()
	if konf.perMinutt == 0 {
		konf.perMinutt, konf.burst = 6000, 1000
	}
	a, err := nyApp(konf, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(a.handler())
	t.Cleanup(s.Close)
	return s
}

func ikkeFolg(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func TestSikkerhetshoder(t *testing.T) {
	s := testServer(t, konfig{})
	for _, sti := range []string{"/", "/finnes-ikke", "/static/stil.css"} {
		res, err := http.Get(s.URL + sti)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options",
			"Referrer-Policy", "Cross-Origin-Opener-Policy"} {
			if res.Header.Get(h) == "" {
				t.Errorf("%s mangler %s", sti, h)
			}
		}
		if res.Header.Get("Strict-Transport-Security") != "" {
			t.Errorf("%s: HSTS skal bare sendes med -hsts", sti)
		}
	}
	if !strings.Contains(mustGet(t, s.URL+"/").Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Error("CSP er ikke streng nok")
	}
}

func mustGet(t *testing.T, u string) *http.Response {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return res
}

func TestHSTSMedFlagg(t *testing.T) {
	s := testServer(t, konfig{hsts: true})
	if mustGet(t, s.URL+"/").Header.Get("Strict-Transport-Security") == "" {
		t.Error("HSTS mangler med -hsts")
	}
}

func TestNyGirRedirectTilGyldigOppgave(t *testing.T) {
	s := testServer(t, konfig{})
	klient := &http.Client{CheckRedirect: ikkeFolg}
	for _, del := range []string{"A", "B", "", "<script>"} {
		res, err := klient.Get(s.URL + "/ny?del=" + url.QueryEscape(del))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusSeeOther {
			t.Fatalf("del=%q: status %d", del, res.StatusCode)
		}
		id := strings.TrimPrefix(res.Header.Get("Location"), "/oppgave/")
		o, err := HentOppgave(id)
		if err != nil {
			t.Fatalf("del=%q: ugyldig ID %q", del, id)
		}
		if (del == "A" || del == "B") && o.Del != del {
			t.Fatalf("del=%q ga del %q", del, o.Del)
		}
	}
}

func TestSjekkRiktigOgGaltSvar(t *testing.T) {
	s := testServer(t, konfig{})
	o, _ := HentOppgave(LagID(0, 42))

	post := func(svar string) string {
		res, err := http.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {svar}, "handling": {"sjekk"}})
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("status %d", res.StatusCode)
		}
		b, _ := io.ReadAll(res.Body)
		return string(b)
	}
	if !strings.Contains(post(strings.ReplaceAll(o.Fasit, "\n", "\r\n")), "Alt riktig") {
		t.Error("riktig svar (med \\r\\n) ble ikke godtatt")
	}
	if strings.Contains(post("feil\nsvar"), "Alt riktig") {
		t.Error("galt svar ble godtatt")
	}
}

func TestSvaretEscapes(t *testing.T) {
	s := testServer(t, konfig{})
	o, _ := HentOppgave(LagID(1, 7))
	res, err := http.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {`<script>alert(1)</script>`}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if strings.Contains(string(b), "<script>alert(1)") {
		t.Error("svaret ble ikke escapet")
	}
}

func TestForStortSkjema(t *testing.T) {
	s := testServer(t, konfig{})
	o, _ := HentOppgave(LagID(2, 1))
	res, err := http.PostForm(s.URL+"/oppgave/"+o.ID, url.Values{"svar": {strings.Repeat("x", 20<<10)}})
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status %d, ville ha 413", res.StatusCode)
	}
}

func TestCrossSitePostAvvises(t *testing.T) {
	s := testServer(t, konfig{})
	o, _ := HentOppgave(LagID(3, 1))
	req, _ := http.NewRequest(http.MethodPost, s.URL+"/oppgave/"+o.ID, strings.NewReader("svar=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, ville ha 403", res.StatusCode)
	}
}

func TestUkjentMetodeOgSti(t *testing.T) {
	s := testServer(t, konfig{})
	req, _ := http.NewRequest(http.MethodDelete, s.URL+"/oppgave/"+LagID(0, 1), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("DELETE: status %d, ville ha 405", res.StatusCode)
	}
	for _, sti := range []string{"/oppgave/1-zzzz", "/oppgave/99-0000000000000001", "/.env", "/static/../main.go"} {
		if kode := mustGet(t, s.URL+sti).StatusCode; kode != http.StatusNotFound && kode != http.StatusMovedPermanently {
			t.Errorf("%s: status %d", sti, kode)
		}
	}
}

func TestRatebegrensning(t *testing.T) {
	s := testServer(t, konfig{perMinutt: 60, burst: 5})
	var sist int
	for i := 0; i < 8; i++ {
		sist = mustGet(t, s.URL+"/").StatusCode
	}
	if sist != http.StatusTooManyRequests {
		t.Errorf("status %d etter 8 raske forespørsler, ville ha 429", sist)
	}
	if kode := mustGet(t, s.URL+"/static/stil.css").StatusCode; kode != http.StatusOK {
		t.Errorf("statiske filer skal ikke begrenses, fikk %d", kode)
	}
}

func TestKlientNokkel(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if k := klientNokkel(r, false); k != "10.0.0.1" {
		t.Errorf("uten proxy: %s", k)
	}
	if k := klientNokkel(r, true); k != "5.6.7.8" {
		t.Errorf("bak proxy: %s", k)
	}
	r.Header.Set("X-Forwarded-For", "tull")
	if k := klientNokkel(r, true); k != "10.0.0.1" {
		t.Errorf("ugyldig XFF: %s", k)
	}
	r.RemoteAddr = "[2001:db8:1:2:3:4:5:6]:443"
	if k := klientNokkel(r, false); k != "2001:db8:1:2::/64" {
		t.Errorf("IPv6: %s", k)
	}
}
