package main

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pdfServer starter en testserver med de gitte PDF-ene (nil = filen mangler).
func pdfServer(t *testing.T, bibel, sporingsark []byte) *httptest.Server {
	t.Helper()
	a, err := nyApp(konfig{perMinutt: 6000, burst: 1000}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	a.bibel.sett(bibel)
	a.sporingsark.sett(sporingsark)
	s := httptest.NewServer(a.handler())
	t.Cleanup(s.Close)
	return s
}

func hentTekst(t *testing.T, u string) (*http.Response, string) {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res, string(b)
}

func TestPDFer(t *testing.T) {
	bibel := []byte("%PDF-1.5\nbibelen\n%%EOF\n")
	ark := []byte("%PDF-1.5\nsporingsark\n%%EOF\n")
	s := pdfServer(t, bibel, ark)

	for fil, pdf := range map[string][]byte{bibelFil: bibel, sporingsarkFil: ark} {
		t.Run(fil, func(t *testing.T) {
			u := s.URL + "/" + fil
			res, body := hentTekst(t, u)
			if res.StatusCode != http.StatusOK || body != string(pdf) {
				t.Fatalf("status %d, fikk %q", res.StatusCode, body)
			}
			for h, vil := range map[string]string{
				"Content-Type":           "application/pdf",
				"Content-Disposition":    `inline; filename="` + fil + `"`,
				"Cache-Control":          "no-cache",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
			} {
				if fikk := res.Header.Get(h); fikk != vil {
					t.Errorf("%s = %q, vil ha %q", h, fikk, vil)
				}
			}
			// Den strenge CSP-en stopper PDF-visningen i nettleseren, så den skal ikke sendes her.
			if csp := res.Header.Get("Content-Security-Policy"); csp != "" {
				t.Errorf("PDF-en skal ikke ha sidenes CSP, fikk %q", csp)
			}
			etag := res.Header.Get("ETag")
			if etag == "" {
				t.Fatal("ETag mangler")
			}

			// Samme versjon: 304 uten innhold.
			req, _ := http.NewRequest(http.MethodGet, u, nil)
			req.Header.Set("If-None-Match", etag)
			res2, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res2.Body.Close()
			if res2.StatusCode != http.StatusNotModified {
				t.Errorf("If-None-Match: status %d, vil ha 304", res2.StatusCode)
			}

			// Range brukes av PDF-visningen i nettleseren.
			req, _ = http.NewRequest(http.MethodGet, u, nil)
			req.Header.Set("Range", "bytes=0-4")
			res3, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := io.ReadAll(res3.Body)
			res3.Body.Close()
			if res3.StatusCode != http.StatusPartialContent || string(b) != "%PDF-" {
				t.Errorf("Range: status %d, fikk %q", res3.StatusCode, b)
			}
		})
	}

	// To ulike filer skal ha ulike ETag-er.
	if mustGet(t, s.URL+"/"+bibelFil).Header.Get("ETag") == mustGet(t, s.URL+"/"+sporingsarkFil).Header.Get("ETag") {
		t.Error("bibelen og sporingsarket har samme ETag")
	}

	// Lenkene vises på forsiden, og bibelen også i menyen på andre sider.
	_, forside := hentTekst(t, s.URL+"/")
	for _, vil := range []string{`href="/frode-bibelen.pdf"`, `href="/sporingsark.pdf"`, "(PDF, 1 kB)", "(PDF, én side, 1 kB)"} {
		if !strings.Contains(forside, vil) {
			t.Errorf("forsiden mangler %s", vil)
		}
	}
	_, feilside := hentTekst(t, s.URL+"/finnes-ikke")
	if !strings.Contains(feilside, `href="/frode-bibelen.pdf"`) {
		t.Error("menyen mangler lenken til bibelen")
	}
}

func TestPDFNyVersjonNyEtag(t *testing.T) {
	etag := func(pdf []byte) string {
		return mustGet(t, pdfServer(t, pdf, nil).URL+"/"+bibelFil).Header.Get("ETag")
	}
	if etag([]byte("%PDF-1.5 a")) == etag([]byte("%PDF-1.5 b")) {
		t.Error("ny PDF skal gi ny ETag")
	}
}

func TestUtenPDFer(t *testing.T) {
	s := pdfServer(t, nil, nil)
	for _, fil := range []string{bibelFil, sporingsarkFil} {
		res, _ := hentTekst(t, s.URL+"/"+fil)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s uten PDF: status %d, vil ha 404", fil, res.StatusCode)
		}
		if res.Header.Get("Content-Security-Policy") == "" {
			t.Errorf("%s: feilsiden skal ha vanlig CSP", fil)
		}
	}
	_, forside := hentTekst(t, s.URL+"/")
	if strings.Contains(forside, ".pdf") || strings.Contains(forside, "Til utskrift") {
		t.Error("forsiden skal ikke lenke til PDF-er som ikke finnes")
	}
}

func TestBareSporingsark(t *testing.T) {
	s := pdfServer(t, nil, []byte("%PDF-1.5 ark"))
	_, forside := hentTekst(t, s.URL+"/")
	if !strings.Contains(forside, `href="/sporingsark.pdf"`) || strings.Contains(forside, "frode-bibelen.pdf") {
		t.Error("forsiden skal lenke til sporingsarket, men ikke til bibelen")
	}
}

// TestInnebygdePDFer sjekker PDF-ene som faktisk er bakt inn (i Docker-byggingen).
func TestInnebygdePDFer(t *testing.T) {
	for _, fil := range []string{bibelFil, sporingsarkFil} {
		b, err := fs.ReadFile(filer, "bibel/"+fil)
		if err != nil {
			t.Logf("ingen %s i bibel/ – bygges i Docker (se bibel/LESMEG.md)", fil)
			continue
		}
		if !bytes.HasPrefix(b, []byte("%PDF-")) {
			t.Errorf("bibel/%s er ikke en PDF", fil)
		}
	}
}
