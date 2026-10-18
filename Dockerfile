# Immagine dell'anteprima dev (ADR 0011): un solo binario, nessun file esterno.
FROM golang:1.27-alpine AS build
ARG BRANCH=locale
ARG COMMIT=sconosciuto
WORKDIR /src
COPY backend/ .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.branch=${BRANCH} -X main.commit=${COMMIT}" -o /gerbillo .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /gerbillo /gerbillo
EXPOSE 8080
ENTRYPOINT ["/gerbillo", "serve"]
