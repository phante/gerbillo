# Immagine dell'anteprima dev (ADR 0011): un solo binario, più git per i commit della
# modifica delle Foto nel Repository dei contenuti, che il Backend crea in /tmp (#43).
FROM golang:1.27-alpine AS build
ARG BRANCH=locale
ARG COMMIT=sconosciuto
WORKDIR /src
COPY backend/ .
RUN CGO_ENABLED=0 go build -trimpath -tags nodynamic \
    -ldflags "-s -w -X main.branch=${BRANCH} -X main.commit=${COMMIT}" -o /gerbillo .

FROM alpine:3.22
RUN apk add --no-cache git
COPY --from=build /gerbillo /gerbillo
# La Gallery pubblica e il seme li genera il workflow prima del build (scripts/anteprima-gallery.sh).
COPY dist/ /dist/
COPY seme/ /seme/
# Il container gira con la radice in sola lettura: git scrive la sua configurazione in /tmp.
ENV HOME=/tmp
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/gerbillo", "serve"]
