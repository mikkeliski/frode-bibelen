# Øvingsside for oppgave 1

En liten nettside for å øve på **«Hva blir utskriften?»** – oppgave 1 på eksamen i PROG1001. Hver gang du trykker «Ny oppgave», lages et nytt C-program i samme stil som på eksamen – og som på eksamen blir utskriften alltid på 5 linjer. Du skriver inn utskriften og får svaret sjekket linje for linje, med fasit og tips etterpå.

- **1A – tekst og pekere:** indekser og hopp, `*t++`, `strchr`/`strrchr`, `strncmp`/`strstr`/`strcmp`, tegnaritmetikk og `toupper`, og å bygge en ny tekst.
- **1B – struct og funksjoner:** struct som kopi eller peker, bytte verdier med referanseoverføring, array av structer, og funksjon i løkke med heltallsdivisjon.

Hver oppgave har en fast lenke (`/oppgave/4-cf536939c6056f92`), så du kan dele en bestemt oppgave med andre.

## Kjør lokalt

Du trenger Go 1.25 eller nyere. Prosjektet bruker bare standardbiblioteket.

```bash
cd oving
go run .                      # åpne http://127.0.0.1:8080
```

På NixOS: `nix-shell -p go --run "go run ."`

| Flagg | Standard | Betydning |
|---|---|---|
| `-adr` | `127.0.0.1:8080` | Adressen serveren lytter på |
| `-bak-proxy` | av | Bruk siste verdi i `X-Forwarded-For` som klient-IP. **Bare** bak en reverse proxy du kontrollerer. |
| `-hsts` | av | Send `Strict-Transport-Security`. **Bare** når siden serveres over HTTPS. |
| `-per-minutt` | `60` | Forespørsler per minutt per klient |
| `-burst` | `30` | Maks forespørsler i en kort topp |

## Sikkerhet

- **Kjører aldri kode.** Hver mal lager både C-programmet og fasiten i Go. Ingen kode blir kompilert eller kjørt på serveren – gcc brukes bare i testene.
- **Ingen data lagres.** Ingen database, ingen cookies, ingen innlogging. En oppgave er helt bestemt av ID-en i lenken.
- **Ingen JavaScript.** Streng `Content-Security-Policy` (`default-src 'none'`), og i tillegg `X-Frame-Options: DENY`, `nosniff`, `Referrer-Policy: same-origin` og `Cross-Origin-Opener-Policy`/`-Resource-Policy`.
- **Cross-site-forespørsler avvises** med `http.CrossOriginProtection`.
- **Alt brukerinnhold escapes** av `html/template`.
- **Strenge grenser:** skjema maks 16 KiB, svar maks 4 KiB, header maks 16 KiB, og tidsavbrudd på lesing, skriving og inaktive forbindelser.
- **Ratebegrensning per IP-adresse** (per /64 for IPv6). Holdes bare i minnet og ryddes fortløpende.
- **Logger ikke IP-adresser** – bare metode, sti, status og tid.
- **Ingen tredjepartsavhengigheter** – bare Go sitt standardbibliotek.
- **Lytter bare på localhost som standard.** Legg den bak en reverse proxy for HTTPS.

## Sett den på nettet

Enkleste oppsett er [Caddy](https://caddyserver.com/) foran, som henter TLS-sertifikat automatisk:

```
oving.example.no {
    reverse_proxy 127.0.0.1:8080
}
```

Start serveren med `-bak-proxy -hsts`. Kjør den gjerne som en låst systemd-tjeneste:

```ini
[Service]
ExecStart=/usr/local/bin/oving -adr 127.0.0.1:8080 -bak-proxy -hsts
DynamicUser=yes
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
CapabilityBoundingSet=
SystemCallFilter=@system-service
MemoryMax=128M
Restart=on-failure
```

Eller med Docker (ikke-root, skrivebeskyttet filsystem, ingen capabilities):

```bash
docker build -t frode-oving .
docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  -p 127.0.0.1:8080:8080 frode-oving
```

## Tester

```bash
go test ./...                    # alt, inkludert gcc-kryssjekk (krever gcc)
go test -short ./...             # uten gcc
OVING_FRO=400 go test -run GCC   # 400 program per mal i stedet for 60
```

`TestFasitMotGCC` kompilerer genererte program med `gcc -Wall -Wextra -Werror`, kjører dem og sjekker at utskriften er **nøyaktig** lik fasiten. De andre testene dekker sikkerhetshoder, størrelsesgrenser, cross-site-POST, escaping, ratebegrensning og svarsjekken.

## Legg til en ny mal

1. Skriv en funksjon `func malNoe(r *rand.Rand) (kode, fasit string, tips []string)` i `maler_a.go` eller `maler_b.go`. Trekk alle tilfeldige verdier fra `r`, så samme ID alltid gir samme oppgave.
2. Lag C-koden i Frodes stil, og regn ut fasiten i Go med nøyaktig samme logikk.
3. Legg malen **til slutt** i `maler` i `oppgaver.go`. Rekkefølgen er en del av ID-formatet, så eldre lenker slutter å virke hvis du flytter på den.
4. Kjør `go test ./...`. gcc-testen avslører med en gang om fasiten ikke stemmer.

## Filer

```
oving/
├── main.go          server, ruter og sikkerhetsmellomvare
├── oppgaver.go      Oppgave-typen, ID-er og felles hjelpere
├── maler_a.go       maler for oppgave 1A
├── maler_b.go       maler for oppgave 1B
├── sjekk.go         sammenligning av svar og fasit
├── begrenser.go     ratebegrensning per klient
├── maler/           HTML-maler (embeddes i programmet)
├── static/          CSS og ikon (embeddes i programmet)
├── *_test.go        tester
└── Dockerfile
```
