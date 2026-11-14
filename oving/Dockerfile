# Bygg:   docker build -t frode-oving .
# Kjør:   docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
#           -v frode-oving-data:/data -p 127.0.0.1:8080:8080 frode-oving
#
# Poengtavla lagres i /data/poeng.json. Bruk et navngitt volum (-v ...:/data),
# ellers forsvinner poengene når containeren lages på nytt.
FROM golang:1.27-alpine AS bygg
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY maler ./maler
COPY static ./static
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
