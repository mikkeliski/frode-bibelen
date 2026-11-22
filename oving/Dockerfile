# Bygges fra ROTMAPPA i repoet, så LaTeX-kildene til bibelen er med:
#   docker build -f oving/Dockerfile -t frode-oving .
# Kjør:   docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
#           -v frode-oving-data:/data -p 127.0.0.1:8080:8080 frode-oving
#
# Poengtavla lagres i /data/poeng.json. Bruk et navngitt volum (-v ...:/data),
# ellers forsvinner poengene når containeren lages på nytt.

# ---- 1) Frode-bibelen og sporingsarket som PDF -----------------------------
# TeX-laget endres bare når pakkelista endres, så det gjenbrukes fra cache.
FROM debian:trixie-slim AS bibel
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      latexmk texlive-latex-recommended texlive-latex-extra \
      texlive-fonts-recommended texlive-pictures texlive-lang-european \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /bibel
# Bare kildefilene kopieres inn – aldri LesData.h. Frodes kode skal ikke ut
# på nettet, og slik kan den ikke komme med selv om den ligger i mappa.
COPY frode-bibelen.tex oppsett.tex forside.tex sporingsark.tex sporingsark-ark.tex ./
COPY kapitler ./kapitler
# Feiler LaTeX, blir det ikke noe nytt image, og den gamle versjonen kjører videre.
RUN latexmk -pdf -interaction=nonstopmode -halt-on-error frode-bibelen.tex sporingsark.tex

# ---- 2) Øvingssiden --------------------------------------------------------
FROM golang:1.27-alpine AS bygg
WORKDIR /src
COPY oving/go.mod ./
COPY oving/*.go ./
COPY oving/maler ./maler
COPY oving/static ./static
COPY oving/bibel ./bibel
COPY --from=bibel /bibel/frode-bibelen.pdf /bibel/sporingsark.pdf ./bibel/
# Testene kjøres i byggingen: feiler en test, blir det ikke noe nytt image,
# og den gamle versjonen kjører videre. (gcc-testen hoppes over med -short.)
RUN CGO_ENABLED=0 go test -short ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /oving .
RUN mkdir -p /data

FROM gcr.io/distroless/static:nonroot
COPY --from=bygg /oving /oving
# Eies av nonroot (65532), slik at et nytt navngitt volum blir skrivbart.
COPY --from=bygg --chown=65532:65532 /data /data
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/oving", "-adr", ":8080", "-poengfil", "/data/poeng.json"]
