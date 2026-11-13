// Kommando oving er en liten, sikker nettside for å øve på tilfeldig
// genererte «Hva blir utskriften?»-oppgaver (oppgave 1 i PROG1001).
//
// Siden kjører aldri kode: hver oppgave og fasiten dens lages i Go fra en
// mal og et frø. Den har ingen database og ingen JavaScript, logger ikke
// IP-adresser, og setter bare en cookie for dem som blir med på poengtavla.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
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

	poengfil     string // "" = poengtavla bare i minnet
	maksSpillere int
}

const (
	cookieNavn       = "spiller"
	maksPoengSkjema  = 4 << 10
	visTopp          = 50
	registreringPer  = 5 * time.Minute // én ny spiller per 5 min per klient ...
	registreringTopp = 5               // ... etter de første 5
)

type app struct {
	sider      map[string]*template.Template
	konf       konfig
	begrenser  *begrenser
	regBegrens *begrenser
	tavle      *poengtavle
	logg       *slog.Logger
}

func nyApp(konf konfig, logg *slog.Logger) (*app, error) {
	if konf.maksSpillere <= 0 {
		konf.maksSpillere = 5_000
	}
	tavle, err := nyPoengtavle(konf.poengfil, konf.maksSpillere)
	if err != nil {
		return nil, fmt.Errorf("poengtavle: %w", err)
	}
	a := &app{
		sider:      map[string]*template.Template{},
		konf:       konf,
		begrenser:  nyBegrenser(konf.perMinutt, konf.burst, 50_000),
		regBegrens: nyBegrenserRate(1/registreringPer.Seconds(), registreringTopp, 50_000),
		tavle:      tavle,
		logg:       logg,
	}
	for _, side := range []string{"forside", "oppgave", "feil", "poeng"} {
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
	mux.HandleFunc("GET /poeng", a.poengside)
	mux.HandleFunc("POST /poeng/bli-med", a.bliMed)
	mux.HandleFunc("POST /poeng/slett", a.slettMeg)
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

	Spiller        *SpillerInfo // nil = ikke med på poengtavla
	AlleredeFerdig bool         // oppgaven gir ikke flere poeng for spilleren
	PoengMelding   string
	FikkPoeng      bool
	Tavle          []Rad
	AntallSpillere int
	NavnFeil       string
	NavnVerdi      string
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
	tok := hentToken(r)
	a.vis(w, r, http.StatusOK, "oppgave", sideData{
		Tittel: "Oppgave 1" + o.Del, Oppgave: &o,
		Spiller: a.tavle.Info(tok), AlleredeFerdig: a.tavle.Ferdig(tok, o.ID),
	})
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
	d := sideData{Tittel: "Oppgave 1" + o.Del, Oppgave: &o, Svar: svar}
	tok := hentToken(r)
	if r.PostForm.Get("handling") == "fasit" {
		d.PoengMelding = a.tavle.FasitVist(tok, o.ID)
		d.VisFasit = true
	} else {
		res := Sjekk(svar, o.Fasit)
		d.Resultat = &res
		if strings.TrimSpace(svar) != "" { // et tomt svar er ikke et forsøk
			var p int
			p, d.PoengMelding = a.tavle.Svar(tok, o.ID, res.AlleRiktig) // poeng FØR fasiten vises
			d.FikkPoeng = p > 0
		}
		// Et galt svar viser bare hvilke linjer som er riktige, ikke fasiten.
		// Ellers kunne man svart tull, lest fasiten og svart riktig etterpå.
		d.VisFasit = res.AlleRiktig
	}
	if d.VisFasit {
		a.tavle.Avslor(o.ID) // gjelder alle, også besøkende uten cookie
	}
	d.Spiller = a.tavle.Info(tok)
	d.AlleredeFerdig = a.tavle.Ferdig(tok, o.ID)
	a.vis(w, r, http.StatusOK, "oppgave", d)
}

// ------------------------------------------------------------ poengtavle ---

func hentToken(r *http.Request) string {
	c, err := r.Cookie(cookieNavn)
	if err != nil {
		return ""
	}
	return c.Value
}

func (a *app) settCookie(w http.ResponseWriter, r *http.Request, verdi string, maksAlder int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNavn,
		Value:    verdi,
		Path:     "/",
		MaxAge:   maksAlder,
		HttpOnly: true,
		Secure:   a.konf.hsts || r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *app) poengside(w http.ResponseWriter, r *http.Request) {
	a.visPoeng(w, r, http.StatusOK, "", "")
}

func (a *app) visPoeng(w http.ResponseWriter, r *http.Request, status int, navnFeil, navnVerdi string) {
	tok := hentToken(r)
	a.vis(w, r, status, "poeng", sideData{
		Tittel: "Poengtavle", Spiller: a.tavle.Info(tok), Tavle: a.tavle.Topp(visTopp, tok),
		AntallSpillere: a.tavle.Antall(), NavnFeil: navnFeil, NavnVerdi: navnVerdi,
	})
}

// lesSkjema leser et lite skjema og svarer selv med feilside hvis det ikke går.
func (a *app) lesSkjema(w http.ResponseWriter, r *http.Request, maks int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maks)
	if err := r.ParseForm(); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			a.feil(w, r, http.StatusRequestEntityTooLarge, "Skjemaet er for stort.")
		} else {
			a.feil(w, r, http.StatusBadRequest, "Klarte ikke å lese skjemaet.")
		}
		return false
	}
	return true
}

func (a *app) bliMed(w http.ResponseWriter, r *http.Request) {
	if !a.lesSkjema(w, r, maksPoengSkjema) {
		return
	}
	if a.tavle.Info(hentToken(r)) != nil { // allerede med
		http.Redirect(w, r, "/poeng", http.StatusSeeOther)
		return
	}
	navn := rensNavn(r.PostForm.Get("navn"))
	if len(navn) > 4*maksNavnTegn {
		navn = navn[:4*maksNavnTegn]
	}
	if gyldigNavn(navn) != nil {
		a.visPoeng(w, r, http.StatusBadRequest, "Kallenavnet må ha 2–20 tegn: bokstaver, tall, mellomrom og - _ .", navn)
		return
	}
	if !a.regBegrens.tillat(klientNokkel(r, a.konf.bakProxy)) {
		w.Header().Set("Retry-After", "300")
		a.feil(w, r, http.StatusTooManyRequests, "For mange nye spillere herfra. Prøv igjen om noen minutter.")
		return
	}
	token, err := a.tavle.BliMed(navn)
	switch {
	case errors.Is(err, ErrNavnOpptatt):
		a.visPoeng(w, r, http.StatusConflict, "Kallenavnet er opptatt. Velg et annet.", navn)
		return
	case errors.Is(err, ErrTavlaFull):
		a.feil(w, r, http.StatusServiceUnavailable, "Poengtavla er full.")
		return
	case err != nil:
		a.logg.Error("kunne ikke lage spiller", "feil", err)
		a.feil(w, r, http.StatusInternalServerError, "Noe gikk galt. Prøv igjen.")
		return
	}
	a.settCookie(w, r, token, 365*24*60*60)
	http.Redirect(w, r, "/poeng", http.StatusSeeOther)
}

func (a *app) slettMeg(w http.ResponseWriter, r *http.Request) {
	if !a.lesSkjema(w, r, maksPoengSkjema) {
		return
	}
	if r.PostForm.Get("bekreft") == "ja" {
		a.tavle.Slett(hentToken(r))
		a.settCookie(w, r, "", -1)
	}
	http.Redirect(w, r, "/poeng", http.StatusSeeOther)
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
		// same-origin (ikke no-referrer): med no-referrer sender nettleseren
		// «Origin: null» på skjemaer over vanlig HTTP, og da avviser
		// CrossOriginProtection våre egne skjemaer.
		h.Set("Referrer-Policy", "same-origin")
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
	poengfil := flag.String("poengfil", "", "fil der poengtavla lagres (tom = bare i minnet)")
	maksSpillere := flag.Int("maks-spillere", 5000, "maks antall spillere på poengtavla")
	slettSpiller := flag.String("slett-spiller", "", "slett spilleren med dette kallenavnet fra -poengfil og avslutt (stopp serveren først)")
	flag.Parse()

	logg := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if *slettSpiller != "" {
		os.Exit(slettFraFil(*poengfil, *slettSpiller, logg))
	}

	a, err := nyApp(konfig{bakProxy: *bakProxy, hsts: *hsts, perMinutt: *perMinutt, burst: *burst,
		poengfil: *poengfil, maksSpillere: *maksSpillere}, logg)
	if err != nil {
		logg.Error("klarte ikke å starte", "feil", err)
		os.Exit(1)
	}
	if *poengfil == "" {
		logg.Warn("poengtavla lagres bare i minnet (bruk -poengfil for å lagre den)")
	} else {
		logg.Info("poengtavla lastet", "fil", *poengfil, "spillere", a.tavle.Antall())
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

	go func() { // lagrer poengtavla hvert 5. sekund hvis noe er endret
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		var forrige string
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := a.tavle.Lagre(); err != nil {
					if err.Error() != forrige { // ikke fyll loggen med samme feil
						logg.Error("klarte ikke å lagre poengtavla", "feil", err)
						forrige = err.Error()
					}
				} else {
					forrige = ""
				}
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
	if err := a.tavle.Lagre(); err != nil {
		logg.Error("klarte ikke å lagre poengtavla", "feil", err)
	}
}

// slettFraFil fjerner en spiller (f.eks. et upassende kallenavn) direkte i fila.
func slettFraFil(fil, navn string, logg *slog.Logger) int {
	if fil == "" {
		logg.Error("-slett-spiller krever -poengfil")
		return 1
	}
	t, err := nyPoengtavle(fil, 1<<30)
	if err != nil {
		logg.Error("klarte ikke å lese poengtavla", "feil", err)
		return 1
	}
	t.mu.Lock()
	h, finnes := t.navn[navneNokkel(rensNavn(navn))]
	if finnes {
		delete(t.navn, navneNokkel(t.spillere[h].Navn))
		delete(t.spillere, h)
		t.endret = true
	}
	t.mu.Unlock()
	if !finnes {
		logg.Error("fant ingen spiller med det kallenavnet", "navn", navn)
		return 1
	}
	if err := t.Lagre(); err != nil {
		logg.Error("klarte ikke å lagre", "feil", err)
		return 1
	}
	logg.Info("spilleren er slettet", "navn", navn)
	return 0
}
