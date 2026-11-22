# bibel/

Her legger Docker-byggingen `frode-bibelen.pdf` og `sporingsark.pdf`, som bygges fra LaTeX-kildene i rotmappa. Filene bakes inn i programmet og serveres på `/frode-bibelen.pdf` og `/sporingsark.pdf`.

Uten PDF-ene virker siden som før, men uten lenkene. Vil du teste dem lokalt, bygger du dem og kopierer dem hit:

```bash
latexmk -pdf frode-bibelen.tex sporingsark.tex      # i rotmappa
cp frode-bibelen.pdf sporingsark.pdf oving/bibel/
```

Har du `LesData.h` i rotmappa, kommer Frodes kode med i PDF-en. Den skal ikke ut på nettet, så Docker-byggingen kopierer bare `.tex`-filene. PDF-er i denne mappa blir aldri committet (`*.pdf` står i `.gitignore`).
