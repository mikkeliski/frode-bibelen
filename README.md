# Frode-bibelen

**Et uoffisielt oppslagsverk i LaTeX til skoleeksamen i PROG1001 Grunnleggende programmering (C) ved NTNU.**

På eksamen i PROG1001 er alle trykte og skrevne hjelpemidler tillatt. Frode-bibelen er laget for å skrives ut, settes i en perm og ligge ved siden av deg i de fire timene. Den samler alt du trenger å slå opp i én bok: maler for oppgave 2 i Frodes stil, feller og metode for «Hva blir utskriften?» i oppgave 1, og en oversikt over alle eksamenene fra H19 til S26.

<p align="center">
  <img src="docs/forside.png" alt="Forsiden til Frode-bibelen" width="360">
</p>

```
           cC
          cC
          cC
           cC
            cCc
              cC
               cC
                cC
                cC
                cC
c               cC
Cc     .-'\    cC
 Cc   /\   \  cC
  Cc. \ \   \cC  .-'\
   `Cc.\ \   \.-'    \
     `Cc\ \        .-/
        .-'     .-'.-'
     .-'        \-'
    /\    .-/\   \
    \ \.-'.-\ \   \
     \/.-'   \ \   \
              \ \   \
               \ \   \
                \ \   \
                 \ \   \
                  \ \   \
                   \ \   \
                    \ \_-/
                     \/-'
```

## Innhold

| Kap. | Tittel | Fil |
|---|---|---|
| 1 | Eksamen: format og strategi | `kapitler/01-eksamen.tex` |
| 2 | `LesData.h` – verktøykassen | `kapitler/02-lesdata.tex` |
| 3 | Programskjelettet i Frodes stil | `kapitler/03-skjelett.tex` |
| 4 | C-grunnlag: oppslag | `kapitler/04-c-grunnlag.tex` |
| 5 | Tekster (`char`-arrayer og `char*`) | `kapitler/05-tekster.tex` |
| 6 | Funksjoner og parametre | `kapitler/06-funksjoner.tex` |
| 7 | Struct, pekere og memory | `kapitler/07-struct-pekere.tex` |
| 8 | Maler: standardoperasjoner i oppgave 2 | `kapitler/08-maler.tex` |
| 9 | Filer | `kapitler/09-filer.tex` |
| 10 | Lenket liste (LIFO/FIFO) | `kapitler/10-lenket-liste.tex` |
| 11 | Oppgave 1: «Hva blir utskriften?» | `kapitler/11-oppgave1.tex` |
| 12 | Eksamensarkivet H19–S26 | `kapitler/12-eksamensarkiv.tex` |
| 13 | Sjekkliste | `kapitler/13-sjekkliste.tex` |
| A | ASCII-tabell (tegn 32–126) | `vedlegg/a-ascii.tex` |
| B | Sporingsark | `vedlegg/b-sporingsark.tex` |

Hvert kapittel starter på ny side, og topptekstene viser hvilket kapittel du er i. Da er det lett å sette inn faneark i permen. Under oppgave 2 er det kapittel 8 og 9 du blar mest i. Under oppgave 1 er det kapittel 11 og vedleggene.

## Bygg PDF-en

### Lokalt

Du trenger en LaTeX-distribusjon, for eksempel [TeX Live](https://tug.org/texlive/) eller [MiKTeX](https://miktex.org/).

```bash
latexmk -pdf frode-bibelen.tex
```

Uten `latexmk` kjører du `pdflatex frode-bibelen.tex` tre ganger. Det trengs før innholdsfortegnelsen får riktige sidetall.

### Overleaf

1. Last ned repoet som zip (**Code → Download ZIP**).
2. Velg **New Project → Upload Project** i Overleaf og last opp zip-filen.
3. Kontroller at kompilatoren er **pdfLaTeX** og at hovedfilen er `frode-bibelen.tex`.

### Pakker

Prosjektet bruker pakkene `listings`, `tcolorbox`, `tikz`, `titlesec`, `fancyhdr`, `enumitem`, `booktabs`, `longtable`, `multicol`, `microtype`, `hyperref`, `pifont` og `amssymb`. Fontene er `charter` (brødtekst), `helvet` (overskrifter), `txfonts` (kode) og Palatino (forsiden). En full TeX Live-installasjon eller Overleaf har alt dette.

Norsk orddeling bruker `babel-norsk` hvis pakken er installert. Mangler den, faller dokumentet tilbake til engelsk orddeling og kompilerer likevel.

## `LesData.h` (ikke inkludert)

`LesData.h` er Frode Haugs egen verktøykasse og ligger **ikke** i dette repoet. Slik får du den med i bibelen:

1. Last ned [`LesData.h`](https://frh.folk.ntnu.no/grprog/eksempler/LesData.h) fra emnesiden.
2. Legg filen i rotmappen, ved siden av `frode-bibelen.tex`.
3. Kompiler på nytt. Kildekoden settes da automatisk inn i kapittel 2, med riktige æ, ø og å.

Uten filen viser kapittel 2 i stedet en boks som forklarer hvor du finner den. `LesData.h` står i `.gitignore`, så den blir ikke pushet ved et uhell.

## Tilpasning

### Forside for utskrift eller skjerm

Forsiden er hvit med gullramme, så den egner seg for utskrift. Vil du heller ha svart skinn og gull, endrer du denne linjen i `oppsett.tex`:

```latex
\morkforsidefalse   % hvit (utskrift)
\morkforsidetrue    % mørk (skjerm/PDF)
```

### Kompiler bare noen kapitler

Fjern `%` foran `\includeonly` i `frode-bibelen.tex` og list opp kapitlene du jobber med:

```latex
\includeonly{kapitler/08-maler,kapitler/09-filer}
```

Sidetall og kryssreferanser beholdes fra forrige fulle kjøring.

### Egne notater

Bruk de samme byggesteinene som resten av bibelen:

```latex
\begin{tips}[Min tittel]          % grønn boks
  ...
\end{tips}

\begin{felle}[Typisk feil]        % rød boks
  ...
\end{felle}

\begin{regel}[Husk]               % blå boks
  ...
\end{regel}

\begin{frode}[Frode sier]         % oransje boks
  ...
\end{frode}

\begin{kode}
int main()  {
  printf("Hei\n");
  return 0;
}
\end{kode}
```

Inneholder tittelen et komma, må den stå i krøllparenteser: `\begin{felle}[{Feller, del 2}]`.

Hver fil i `kapitler/` og `vedlegg/` starter med `% !TEX root = ../frode-bibelen.tex`. Da kompilerer TeXstudio og VS Code (LaTeX Workshop) hovedfilen selv om du har et kapittel åpent.

## Struktur

```
frode-bibelen/
├── frode-bibelen.tex      hovedfil – kompiler denne
├── oppsett.tex            pakker, farger, kodestil, bokser, topptekst
├── forside.tex            forside og kolofon
├── kapitler/              ett kapittel per fil (01–13)
├── vedlegg/               ASCII-tabell og sporingsark
├── docs/forside.png       forhåndsvisning til README
└── LesData.h              (legg til selv – se over)
```

## Bidra

Rettelser og forbedringer er velkomne som issues eller pull requests. Hold deg til disse retningslinjene:

- **Skriv kode i Frodes stil:** `g`-prefiks på globale variable, `struct`-funksjoner som heter `<struct><Verb>` og `const` på parametre som bare leses. Bruk `Kunde` som eksempel i malene.
- **Test oppgave 1-eksempler:** kompiler og kjør koden, og sjekk at fasiten i teksten stemmer med det programmet faktisk skriver ut.
- **Ikke kopier Frodes materiale:** ikke lim inn kode fra løsningsforslag, `LesData.h` eller oppgavetekster. Skriv egne maler, og lenk til originalene.
- **Kompiler før pull request:** hele dokumentet skal bygge uten feil eller `Overfull`-advarsler.

## Kilder og ansvarsfraskrivelse

Innholdet bygger på Frode Haugs emneside for PROG1001: [frh.folk.ntnu.no/grprog](https://frh.folk.ntnu.no/grprog/). Derfra er opplegget, [eksamensrådene](https://frh.folk.ntnu.no/grprog/ex_raad.html), stikkordsoppsummeringen, eksemplene (EKS_01–EKS_39) og [eksamensarkivet](https://frh.folk.ntnu.no/grprog/eksamen/) med eksamenene H19–S26 brukt.

Frode-bibelen er et **uoffisielt, studentlaget hjelpemiddel**. Den er verken laget eller godkjent av emneansvarlig eller NTNU. Kodemalene er egne maler skrevet i Frodes stil. `LesData.h`, oppgavetekstene og løsningsforslagene tilhører opphavspersonene og er ikke inkludert, bare lenket til.

Sjekk alltid gjeldende eksamensregler for ditt semester: format, vekting og tillatte hjelpemidler kan endre seg.
