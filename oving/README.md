# Øvingsside for oppgave 1

En liten nettside for å øve på **«Hva blir utskriften?»** – oppgave 1 på eksamen i PROG1001. Hver gang du trykker «Ny oppgave», lages et nytt C-program i samme stil som på eksamen. Du skriver inn utskriften og får svaret sjekket linje for linje, med fasit og tips etterpå.

- **1A – tekst og pekere:** indekser og hopp, `*t++`, `strchr`/`strrchr`, `strncmp`/`strstr`/`strcmp`, tegnaritmetikk og `toupper`, og å bygge en ny tekst.
- **1B – struct og funksjoner:** struct som kopi eller peker, bytte verdier med referanseoverføring, array av structer, og funksjon i løkke med heltallsdivisjon.

Hver oppgave har en fast lenke (`/oppgave/4-cf536939c6056f92`), så du kan dele en bestemt oppgave med andre.

## Poengtavle

Velg et kallenavn på `/poeng` for å samle poeng: **3** for riktig svar på første forsøk, **2** på andre og **1** deretter. Hver oppgave gir poeng bare én gang. Tomme svar teller ikke som forsøk.

Et galt svar viser bare hvilke linjer som er riktige (✓/✗), ikke fasiten. Fasiten vises først når svaret er riktig eller noen trykker «Vis fasit», og fra da av gir oppgaven ingen poeng til noen som ikke allerede har løst den. Det gjelder også om fasiten ble vist i et privat vindu, en annen nettleser eller for en annen konto. Tavla viser de 50 beste, og deg selv hvis du er lenger ned.

Tavla bygger på ærlighet – koden kan jo alltid kjøres på egen maskin.

- **Frivillig:** bare de som blir med, får en cookie. Den er `HttpOnly` og `SameSite=Lax`, og `Secure` med `-hsts`.
- **Ingen passord:** cookien inneholder en tilfeldig nøkkel på 256 bit. Serveren lagrer bare SHA-256 av nøkkelen, så poengfila kan ikke brukes til å utgi seg for noen.
- **Kallenavn:** 2–20 tegn (bokstaver inkludert æøå, tall, mellomrom og `- _ .`), unike uten hensyn til store/små bokstaver.
- **Begrenset:** maks én ny spiller per 5. minutt per IP etter de 5 første, og maks `-maks-spillere` totalt.
- **Slett meg:** spillerne kan slette seg selv og poengene sine fra tavla.
- **Lagring:** en JSON-fil (`-poengfil`) som skrives atomisk hvert 5. sekund ved endringer og når serveren avslutter. Uten `-poengfil` ligger tavla bare i minnet.

Upassende kallenavn kan du bytte eller slette. Stopp serveren først, ellers skriver den over fila. Ved navnebytte beholder spilleren poengene sine og er fortsatt logget inn.

```bash
sudo docker stop oving
sudo docker run --rm -v frode-oving-data:/data frode-oving -bytt-navn "Gammelt navn" -nytt-navn "Nytt navn"
sudo docker run --rm -v frode-oving-data:/data frode-oving -slett-spiller "Kallenavn"
sudo docker start oving
```

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
| `-poengfil` | tom | Fil der poengtavla lagres. Tom = bare i minnet. Docker-imaget bruker `/data/poeng.json`. |
| `-maks-spillere` | `5000` | Maks antall spillere på poengtavla |
| `-slett-spiller` | | Slett spilleren med dette kallenavnet fra `-poengfil` og avslutt |
| `-bytt-navn`, `-nytt-navn` | | Gi spilleren med kallenavnet i `-bytt-navn` navnet i `-nytt-navn` i `-poengfil`, og avslutt |

## Sikkerhet

- **Kjører aldri kode.** Hver mal lager både C-programmet og fasiten i Go. Ingen kode blir kompilert eller kjørt på serveren – gcc brukes bare i testene.
- **Minimalt med data.** Ingen database og ingen innlogging. En oppgave er helt bestemt av ID-en i lenken. Det eneste som lagres, er poengtavla for dem som velger å bli med (se over).
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
ExecStart=/usr/local/bin/oving -adr 127.0.0.1:8080 -bak-proxy -hsts -poengfil /var/lib/frode-oving/poeng.json
DynamicUser=yes
StateDirectory=frode-oving
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
  -v frode-oving-data:/data -p 127.0.0.1:8080:8080 frode-oving
```

Poengtavla ligger i volumet `frode-oving-data` og overlever at containeren slettes og bygges på nytt. Sikkerhetskopi: `sudo docker run --rm -v frode-oving-data:/data -v "$PWD":/ut busybox cp /data/poeng.json /ut/`.

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
├── poengtavle.go    poengtavla og lagringen av den
├── maler/           HTML-maler (embeddes i programmet)
├── static/          CSS og ikon (embeddes i programmet)
├── *_test.go        tester
└── Dockerfile
```
