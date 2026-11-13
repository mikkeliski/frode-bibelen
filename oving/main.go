// Kommando oving er en liten, sikker nettside for å øve på tilfeldig
// genererte «Hva blir utskriften?»-oppgaver (oppgave 1 i PROG1001).
//
// Siden kjører aldri kode: hver oppgave og fasiten dens lages i Go fra en
// mal og et frø. Den har ingen database, ingen cookies, ingen JavaScript og
// logger ikke IP-adresser.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

//go:embed maler/*.html static/*
var filer embed.FS

const (
	maksSkjemaBytes = 16 << 10 // maks størrelse på et innsendt skjema
	maksSvarBytes   = 4 << 10  // maks størrelse på selve svaret
)

type konfig struct {
	bakProxy  bool // stol på X-Forwarded-For (bare bak en reverse proxy)
	hsts      bool // send Strict-Transport-Security (bare over HTTPS)
	perMinutt int  // forespørsler per minutt per klient
	burst     int  // korte topper
}

type app struct {
	sider     map[string]*template.Template
	konf      konfig
	begrenser *begrenser
	logg      *slog.Logger
}

func nyApp(konf konfig, logg *slog.Logger) (*app, error) {
	a := &app{
		sider:     map[string]*template.Template{},
		konf:      konf,
		begrenser: nyBegrenser(konf.perMinutt, konf.burst, 50_000),
		logg:      logg,
	}
	for _, side := range []string{"forside", "oppgave", "feil"} {
		t, err := template.ParseFS(filer, "maler/base.html", "maler/"+side+".html")
		if err != nil {
			return nil, err
		}
		a.sider[side] = t
	}
	return a, nil
}

// handler bygger hele kjeden: logging -> sikkerhetshoder -> ratebegrensning ->
// beskyttelse mot cross-site-forespørsler -> rutene.
func (a *app) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.forside)
	mux.HandleFunc("GET /ny", a.ny)
	mux.HandleFunc("GET /oppgave/{id}", a.visOppgave)
	mux.HandleFunc("POST /oppgave/{id}", a.sjekkSvar)
	mux.HandleFunc("GET /helse", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	static, _ := fs.Sub(filer, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", statiske(http.FileServerFS(static))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodPost:
			a.feil(w, r, http.StatusNotFound, "Siden finnes ikke.")
		default:
			w.Header().Set("Allow", "GET, HEAD, POST")
			a.feil(w, r, http.StatusMethodNotAllowed, "Metoden er ikke tillatt.")
		}
	})

	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.feil(w, r, http.StatusForbidden, "Forespørselen kom fra en annen nettside og ble avvist.")
	}))

	return a.loggForesporsel(a.sikkerhetshoder(a.begrens(cop.Handler(mux))))
}

// ---------------------------------------------------------------- sider ---

type sideData struct {
	Tittel   string
	Oppgave  *Oppgave
	Svar     string
	Resultat *Resultat
	VisFasit bool
	Melding  string
	MalerA   []string
	MalerB   []string
}

func (a *app) forside(w http.ResponseWriter, r *http.Request) {
	d := sideData{Tittel: "Øv på oppgave 1"}
	for _, m := range maler {
		if m.del == "A" {
			d.MalerA = append(d.MalerA, m.tittel)
		} else {
			d.MalerB = append(d.MalerB, m.tittel)
		}
	}
	a.vis(w, r, http.StatusOK, "forside", d)
}

func (a *app) ny(w http.ResponseWriter, r *http.Request) {
	del := r.URL.Query().Get("del")
	if del != "A" && del != "B" {
		del = ""
	}
	http.Redirect(w, r, "/oppgave/"+NyID(del), http.StatusSeeOther)
}

func (a *app) visOppgave(w http.ResponseWriter, r *http.Request) {
	o, err := HentOppgave(r.PathValue("id"))
	if err != nil {
		a.feil(w, r, http.StatusNotFound, "Fant ingen oppgave med denne lenken.")
		return
	}
	a.vis(w, r, http.StatusOK, "oppgave", sideData{Tittel: "Oppgave 1" + o.Del, Oppgave: &o})
}

func (a *app) sjekkSvar(w http.ResponseWriter, r *http.Request) {
	o, err := HentOppgave(r.PathValue("id"))
	if err != nil {
		a.feil(w, r, http.StatusNotFound, "Fant ingen oppgave med denne lenken.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maksSkjemaBytes)
	if err := r.ParseForm(); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			a.feil(w, r, http.StatusRequestEntityTooLarge, "Svaret er for langt.")
			return
		}
		a.feil(w, r, http.StatusBadRequest, "Klarte ikke å lese skjemaet.")
		return
	}
	svar := r.PostForm.Get("svar")
	if len(svar) > maksSvarBytes || !utf8.ValidString(svar) {
		a.feil(w, r, http.StatusBadRequest, "Svaret er for langt eller inneholder ugyldige tegn.")
		return
	}
	d := sideData{Tittel: "Oppgave 1" + o.Del, Oppgave: &o, Svar: svar, VisFasit: true}
	if r.PostForm.Get("handling") != "fasit" {
		res := Sjekk(svar, o.Fasit)
		d.Resultat = &res
	}
	a.vis(w, r, http.StatusOK, "oppgave", d)
}

func (a *app) feil(w http.ResponseWriter, r *http.Request, status int, melding string) {
	a.vis(w, r, status, "feil", sideData{Tittel: http.StatusText(status), Melding: melding})
}

// vis rendrer til en buffer først, slik at en malfeil aldri gir en halv side.
func (a *app) vis(w http.ResponseWriter, r *http.Request, status int, side string, d sideData) {
	var buf bytes.Buffer
	if err := a.sider[side].ExecuteTemplate(&buf, "base", d); err != nil {
		a.logg.Error("malfeil", "side", side, "feil", err)
		http.Error(w, "Intern feil", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = buf.WriteTo(w)
	}
}

// ------------------------------------------------------------ mellomvare ---

func (a *app) sikkerhetshoder(neste http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self'; "+
			"form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		if a.konf.hsts {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		neste.ServeHTTP(w, r)
	})
}

func (a *app) begrens(neste http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static/") && !a.begrenser.tillat(klientNokkel(r, a.konf.bakProxy)) {
			w.Header().Set("Retry-After", "10")
			a.feil(w, r, http.StatusTooManyRequests, "For mange forespørsler. Vent litt og prøv igjen.")
			return
		}
		neste.ServeHTTP(w, r)
	})
}

type statusSkriver struct {
	http.ResponseWriter
	status int
}

func (s *statusSkriver) WriteHeader(kode int) {
	s.status = kode
	s.ResponseWriter.WriteHeader(kode)
}

// loggForesporsel logger metode, sti, status og tid - men aldri IP-adresser.
func (a *app) loggForesporsel(neste http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusSkriver{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if v := recover(); v != nil {
				a.logg.Error("panikk", "sti", r.URL.Path, "verdi", v)
				http.Error(sw, "Intern feil", http.StatusInternalServerError)
			}
			a.logg.Info("forespørsel", "metode", r.Method, "sti", r.URL.Path,
				"status", sw.status, "ms", time.Since(start).Milliseconds())
		}()
		neste.ServeHTTP(sw, r)
	})
}

func statiske(neste http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		neste.ServeHTTP(w, r)
	})
}

// ------------------------------------------------------------------ main ---

func main() {
	adr := flag.String("adr", "127.0.0.1:8080", "adressen serveren lytter på")
	bakProxy := flag.Bool("bak-proxy", false, "stol på X-Forwarded-For (bruk BARE bak en reverse proxy)")
	hsts := flag.Bool("hsts", false, "send Strict-Transport-Security (bruk BARE når siden serveres over HTTPS)")
	perMinutt := flag.Int("per-minutt", 60, "forespørsler per minutt per klient")
	burst := flag.Int("burst", 30, "maks forespørsler i en kort topp")
	flag.Parse()

	logg := slog.New(slog.NewTextHandler(os.Stderr, nil))
	a, err := nyApp(konfig{bakProxy: *bakProxy, hsts: *hsts, perMinutt: *perMinutt, burst: *burst}, logg)
	if err != nil {
		logg.Error("klarte ikke å laste maler", "feil", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              *adr,
		Handler:           a.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(logg.Handler(), slog.LevelWarn),
	}

	ctx, stopp := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopp()

	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				a.begrenser.rydd()
			}
		}
	}()

	go func() {
		logg.Info("øvingssiden kjører", "adresse", "http://"+*adr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logg.Error("serverfeil", "feil", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logg.Info("avslutter")
	avslutt, avbryt := context.WithTimeout(context.Background(), 10*time.Second)
	defer avbryt()
	_ = srv.Shutdown(avslutt)
}
