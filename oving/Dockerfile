# Bygg:   docker build -t frode-oving .
# Kjør:   docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
#           -p 127.0.0.1:8080:8080 frode-oving
FROM golang:1.27-alpine AS bygg
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY maler ./maler
COPY static ./static
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /oving .

FROM gcr.io/distroless/static:nonroot
COPY --from=bygg /oving /oving
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/oving", "-adr", ":8080"]
