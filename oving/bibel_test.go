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

// bibelServer starter en testserver med pdf som bibel (nil = uten PDF).
func bibelServer(t *testing.T, pdf []byte) *httptest.Server {
	t.Helper()
	a, err := nyApp(konfig{perMinutt: 6000, burst: 1000}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	a.settBibel(pdf)
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

func TestBibelPDF(t *testing.T) {
	pdf := []byte("%PDF-1.5\nbare en test\n%%EOF\n")
	s := bibelServer(t, pdf)

	res, body := hentTekst(t, s.URL+"/frode-bibelen.pdf")
	if res.StatusCode != http.StatusOK || body != string(pdf) {
		t.Fatalf("status %d, fikk %q", res.StatusCode, body)
	}
	for h, vil := range map[string]string{
		"Content-Type":           "application/pdf",
		"Content-Disposition":    `inline; filename="frode-bibelen.pdf"`,
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
	req, _ := http.NewRequest(http.MethodGet, s.URL+"/frode-bibelen.pdf", nil)
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
	req, _ = http.NewRequest(http.MethodGet, s.URL+"/frode-bibelen.pdf", nil)
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

	// Lenkene vises på forsiden og i menyen på andre sider.
	_, forside := hentTekst(t, s.URL+"/")
	if !strings.Contains(forside, `href="/frode-bibelen.pdf"`) || !strings.Contains(forside, "(PDF, 1 kB)") {
		t.Error("forsiden mangler lenken til PDF-en")
	}
	_, feilside := hentTekst(t, s.URL+"/finnes-ikke")
	if !strings.Contains(feilside, `href="/frode-bibelen.pdf"`) {
		t.Error("menyen mangler lenken til PDF-en")
	}
}

func TestBibelPDFNyVersjonNyEtag(t *testing.T) {
	a := mustGetEtag(t, []byte("%PDF-1.5 a"))
	b := mustGetEtag(t, []byte("%PDF-1.5 b"))
	if a == b {
		t.Error("ny PDF skal gi ny ETag")
	}
}

func mustGetEtag(t *testing.T, pdf []byte) string {
	t.Helper()
	return mustGet(t, bibelServer(t, pdf).URL+"/frode-bibelen.pdf").Header.Get("ETag")
}

func TestUtenBibelPDF(t *testing.T) {
	s := bibelServer(t, nil)
	res, _ := hentTekst(t, s.URL+"/frode-bibelen.pdf")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("uten PDF: status %d, vil ha 404", res.StatusCode)
	}
	if res.Header.Get("Content-Security-Policy") == "" {
		t.Error("feilsiden skal ha vanlig CSP")
	}
	_, forside := hentTekst(t, s.URL+"/")
	if strings.Contains(forside, "frode-bibelen.pdf") {
		t.Error("forsiden skal ikke lenke til en PDF som ikke finnes")
	}
}

// TestInnebygdBibel sjekker PDF-en som faktisk er bakt inn (i Docker-byggingen).
func TestInnebygdBibel(t *testing.T) {
	b, err := fs.ReadFile(filer, bibelFil)
	if err != nil {
		t.Skip("ingen PDF i bibel/ – bygges i Docker (se bibel/LESMEG.md)")
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("%s er ikke en PDF", bibelFil)
	}
}
