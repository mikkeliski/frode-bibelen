# bibel/

Her legger Docker-byggingen `frode-bibelen.pdf`, som bygges fra LaTeX-kildene i rotmappa. Filen bakes inn i programmet og serveres på `/frode-bibelen.pdf`.

Uten PDF-en virker siden som før, men uten lenken til bibelen. Vil du teste PDF-en lokalt, bygger du bibelen og kopierer den hit:

```bash
latexmk -pdf frode-bibelen.tex        # i rotmappa
cp frode-bibelen.pdf oving/bibel/
```

Har du `LesData.h` i rotmappa, kommer Frodes kode med i PDF-en. Den skal ikke ut på nettet, så Docker-byggingen kopierer bare `.tex`-filene. PDF-er i denne mappa blir aldri committet (`*.pdf` står i `.gitignore`).
