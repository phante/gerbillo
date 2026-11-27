---
name: Gallery ProgettoDighe
description: La gallery fotografica della community di ProgettoDighe, con il Backend di redazione.
colors:
  calcestruzzo: "#333333"
  nero-quasi: "#111111"
  giallo-cantiere: "#ffd700"
  giallo-segnale: "#ffee00"
  fondo-pagina: "#f4f4f4"
  grigio-chiaro: "#eeeeee"
  bianco: "#ffffff"
  testo: "rgba(0, 0, 0, 0.8)"
  testo-secondario: "rgba(0, 0, 0, 0.6)"
  bordo: "#dddddd"
  bordo-campo: "#bbbbbb"
  esito-ok: "#ddffdd"
  esito-errore: "#ffdddd"
  esito-avviso: "#fff6bf"
  interno-fondo: "#ffe6cc"
  interno-testo: "#8a4b00"
  guasto: "#bb0000"
typography:
  headline:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'avenir next', avenir, 'helvetica neue', helvetica, ubuntu, roboto, noto, 'segoe ui', arial, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 400
    lineHeight: 1.25
    letterSpacing: "0.1em"
  title:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'avenir next', avenir, 'helvetica neue', helvetica, ubuntu, roboto, noto, 'segoe ui', arial, sans-serif"
    fontSize: "1rem"
    fontWeight: 700
    lineHeight: 1.25
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'avenir next', avenir, 'helvetica neue', helvetica, ubuntu, roboto, noto, 'segoe ui', arial, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, 'avenir next', avenir, 'helvetica neue', helvetica, ubuntu, roboto, noto, 'segoe ui', arial, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.25
    letterSpacing: "0.1em"
  mono:
    fontFamily: "monospace, monospace"
    fontSize: "0.8rem"
rounded:
  xs: "0.125rem"
  sm: "0.25rem"
  md: "0.5rem"
  lg: "1rem"
  pill: "999px"
spacing:
  "1": "0.25rem"
  "2": "0.5rem"
  "3": "1rem"
  "4": "2rem"
  "5": "4rem"
components:
  button-primary:
    backgroundColor: "{colors.calcestruzzo}"
    textColor: "{colors.bianco}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 1rem"
  button-moderatore:
    backgroundColor: "{colors.giallo-cantiere}"
    textColor: "{colors.testo}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0.5rem 1rem"
  breadcrumb:
    backgroundColor: "{colors.calcestruzzo}"
    textColor: "{colors.bianco}"
    padding: "0.5rem"
  link-metadato-hover:
    backgroundColor: "{colors.giallo-cantiere}"
    textColor: "{colors.testo}"
    rounded: "{rounded.xs}"
  chip-filtro:
    backgroundColor: "{colors.fondo-pagina}"
    textColor: "#666666"
    rounded: "{rounded.pill}"
    padding: "0.4rem 0.8rem"
  chip-filtro-attivo:
    backgroundColor: "{colors.calcestruzzo}"
    textColor: "{colors.bianco}"
    rounded: "{rounded.pill}"
  backend-pulsante:
    backgroundColor: "{colors.calcestruzzo}"
    textColor: "{colors.bianco}"
    rounded: "4px"
    padding: "0.5rem 1rem"
  backend-pulsante-secondario:
    backgroundColor: "{colors.bianco}"
    textColor: "{colors.calcestruzzo}"
    rounded: "4px"
    padding: "0.5rem 1rem"
  backend-campo:
    backgroundColor: "{colors.bianco}"
    rounded: "4px"
    padding: "0.5rem"
  backend-riquadro:
    backgroundColor: "{colors.bianco}"
    rounded: "6px"
    padding: "1rem"
---

# Design System: Gallery ProgettoDighe

## Overview

**Creative North Star: "L'archivio di cantiere"**

La Gallery è un archivio tecnico, non una vetrina. Le Foto sono documenti di cantiere: stanno su un fondo grigio chiaro come fogli su un tavolo da disegno, ognuna con la sua scheda di dati ordinata (Ritrae, Autore, Scattata, Licenza…). L'interfaccia parla con etichette in maiuscolo spaziato, come le intestazioni di un registro, e con il grigio del calcestruzzo. Il colore vero sta nelle fotografie.

Il sistema è ereditato dal Sito principale: Tachyons, `main.css`, `map.css` e `search.css` sono copiati byte per byte da progettodighe.it (`backend/modelli/sito/`), e la Gallery aggiunge poche regole proprie in `backend/modelli/stile.css`. Il Backend (`backend/modelli/backend/backend.css`) usa gli stessi colori e lo stesso font ma un layout da strumento: barra fissa in alto, riquadri bianchi, azioni fisse in basso, pensato prima per il telefono. Non deve sembrare una sezione del sito.

Il riferimento al Sito principale è quello attuale ma rivedibile: un'evoluzione visiva della Gallery è possibile, e passa da una revisione esplicita di questo file.

**Key Characteristics:**
- Quasi monocromo: calcestruzzo, bianco e grigi; un solo accento, il giallo cantiere.
- Etichette e titoli in maiuscolo spaziato (0.1em).
- Font di sistema, nessun web font.
- Piatto: un'ombra morbida solo sulle Foto e sulle schede che si toccano.
- Pagine strette (max 64rem), griglie di miniature quadrate o 4:3.

## Colors

Una tavolozza di cantiere: grigi da calcestruzzo, carta bianca e un giallo che compare solo dove si interagisce.

### Primary
- **Grigio calcestruzzo** (#333333): il colore delle azioni. Pulsanti pieni («Copia per il forum», «Mostra altre», paginazione), briciole di percorso, chip attivi, bordo della barra di selezione nel Backend.

### Secondary
- **Giallo cantiere** (#ffd700): l'evidenziatore. Fondo dei link dei metadati al passaggio del mouse, pulsante «Modifica» riservato ai Moderatori, link nei popup della mappa.
- **Giallo segnale** (#ffee00): solo nel Backend, per l'anello del pulsante da non perdere (`.pulsante-evidente`) e i numeri dei chip attivi.

### Neutral
- **Fondo pagina** (#f4f4f4): fondo di tutte le pagine pubbliche e della barra delle azioni del Backend.
- **Grigio chiaro** (#eeeeee): segnaposto delle miniature mentre caricano, fondo del piede (sfumato da bianco a #c8c8c8).
- **Bianco** (#ffffff): header, riquadri e campi del Backend, schede «Oppure».
- **Testo** (rgba(0,0,0,.8)): testo principale e titoli.
- **Testo secondario** (rgba(0,0,0,.6)): sottotitoli, conteggi, etichette di sezione, note.
- **Bordo** (#dddddd) e **Bordo campo** (#bbbbbb): divisori, bordi di header e barre; bordi di campi, chip e popup del Backend.

### Stati (solo Backend)
- **Esito ok** (#ddffdd), **Esito errore** (#ffdddd), **Esito avviso** (#fff6bf): fondi dei messaggi di esito. L'avviso è anche il fondo dei suggerimenti e dei chip nuovi.
- **Interno** (fondo #ffe6cc, testo #8a4b00): tag e chip interni, cioè Informazioni interne che non vanno mai nella Gallery pubblica.
- **Guasto** (#bb0000): operazioni fallite nel widget della barra.

### Named Rules
**La regola del giallo raro.** Il giallo cantiere segnala un'interazione: il passaggio del mouse, o un'azione riservata. Non è un colore di sfondo, di sezione o di decorazione.

**La regola dell'arancio interno.** Il colore Interno compare solo nel Backend. Se lo vedi in `dist/`, è un'Informazione interna trapelata.

## Typography

**Display Font:** nessuno: i titoli usano lo stesso font del testo.
**Body Font:** font di sistema (-apple-system, BlinkMacSystemFont, avenir next, helvetica neue, …, sans-serif)
**Label/Mono Font:** monospace solo per i dati grezzi (EXIF, nomi tecnici) nel Backend.

**Character:** Un solo font di sistema, gerarchia fatta con dimensione, peso e maiuscolo spaziato. Sobrio come un registro tecnico.

### Hierarchy
- **Headline** (400, 1.5rem, 1.25, maiuscolo, spaziatura 0.1em): il titolo `h1` di ogni pagina pubblica.
- **Section label** (400, 0.875rem, maiuscolo, spaziatura 0.1em, testo secondario): intestazioni di sezione come «Foto», «Esplora intorno», «Oppure», i nomi delle faccette.
- **Title** (700, 1rem, 1.25): nome di un'Entità sotto la miniatura, nomi nelle schede.
- **Body** (400, 1rem, 1.5): descrizione della Foto (1.25rem da 30em in su, larghezza massima 34em), testo corrente.
- **Label** (400, 0.875rem, maiuscolo, spaziatura 0.1em): nomi dei metadati («Ritrae:», «Autore:»), testo dei pulsanti (in grassetto).
- **Micro** (0.75rem, maiuscolo spaziato, rgba(0,0,0,.5)): la sezione sopra il nome nelle schede «Esplora intorno» e nei risultati di ricerca.

### Named Rules
**La regola del maiuscoletto.** Ciò che etichetta (sezioni, metadati, pulsanti, briciole) va in maiuscolo con spaziatura 0.1em. Ciò che si legge (nomi, descrizioni, valori) resta in minuscolo normale.

## Layout

Pagine pubbliche in una colonna centrata larga al massimo 64rem (`mw8`), con margini laterali di 1rem. La scala di spazi è quella di Tachyons: 0.25, 0.5, 1, 2, 4rem. I punti di rottura sono quelli di Tachyons: 30em (`-ns`) e 60em (`-l`).

- **Griglia delle Foto:** miniature quadrate, 3 per riga sul telefono, 4 da 30em, 5 da 60em, con 0.25rem di spazio.
- **Griglia delle Entità:** miniature quadrate con nome e conteggio sotto, 2, 3 e 5 per riga.
- **Esplora intorno:** una striscia orizzontale di schede 4:3 larghe 9.5rem, che scorre con lo scatto.
- **Scheda della Foto:** l'immagine (max 80vh), la descrizione, poi la griglia dei metadati a tre colonne (icona 24px, etichetta 100px, valore); sotto 30em l'etichetta sparisce e resta l'icona.
- **Backend:** contenuto largo al massimo 64rem, barra fissa in alto che va a capo, azioni fisse in basso, campi a tutta larghezza. Griglia di miniature 4:3 con colonne di almeno 9rem. Filtri in una griglia di colonne da 14rem.

## Elevation & Depth

Piatto per impostazione predefinita. La profondità viene da fondi a toni (grigio pagina, bianco delle schede) e da bordi sottili. L'unica ombra ricorrente sta sotto le Foto e le schede toccabili, come carta appoggiata su un tavolo.

### Shadow Vocabulary
- **Foglio** (`box-shadow: 2px 2px 8px 0 rgba(0,0,0,.2)`): miniature delle Foto, immagine a schermo, schede «Oppure».
- **Popup** (`box-shadow: 0 4px 12px rgba(0,0,0,.12)`): popup del Backend.
- **Barra in basso** (`box-shadow: 0 -2px 8px rgba(0,0,0,.1)`): barra di selezione del Backend.

### Named Rules
**La regola del foglio.** L'ombra è per le fotografie e per ciò che sta sopra la pagina (popup, barre). Pulsanti, chip e intestazioni restano piatti.

## Shapes

Angoli appena smussati, che crescono con la dimensione: 0.125rem sui link evidenziati, 0.25rem su pulsanti e miniature delle Foto, 0.5rem su schede 4:3 e mappa, 1rem sulle miniature delle Entità e sull'immagine a schermo. I chip dei filtri sono a pillola. Le briciole di percorso sono rettangoli pieni senza angoli smussati. Nel Backend: 4px per campi e pulsanti, 6px per riquadri e popup.

## Components

### Buttons
Sobri e decisi: pieni, scuri, testo bianco in maiuscolo spaziato.
- **Shape:** angoli leggermente smussati (0.25rem).
- **Primary:** fondo calcestruzzo, testo bianco grassetto 0.875rem in maiuscolo spaziato, padding 0.5rem 1rem, senza bordo. Icona a sinistra quando c'è.
- **Hover / Focus:** si schiariscono (opacità 0.5 al passaggio, `dim` di Tachyons). Il focus è quello del browser.
- **Moderatore:** come il primario, ma fondo giallo cantiere e testo scuro. Solo «Modifica», visibile solo ai Moderatori.
- **Backend:** primario calcestruzzo con bordo #333 e radius 4px. Il secondario è bianco con testo e bordo calcestruzzo. L'evidente è nero con anello giallo segnale di 3px. La variante piccola ha padding 0.15rem 0.5rem.

### Chips
- **Style:** pillola, fondo #f4f4f4, bordo #ddd, testo #666 0.7rem in grassetto, contatore in una piccola pillola rgba(0,0,0,.06).
- **State:** attivo con fondo calcestruzzo, testo bianco e una × per togliere il filtro. Al passaggio del mouse il fondo passa a #e0e0e0.
- **Backend:** chip dei campi (Tom Select) con fondo #eee e bordo #ccc. Gli interni sono arancio Interno, i nuovi giallo avviso.

### Cards / Containers
- **Corner Style:** 0.5–1rem nelle pagine pubbliche, 6px nel Backend.
- **Background:** nessuno per le schede di Entità (miniatura e testo sul fondo pagina); bianco per le schede «Oppure» e i riquadri del Backend.
- **Shadow Strategy:** ombra «Foglio» sulle schede pubbliche toccabili, nessuna sui riquadri del Backend (bordo #e5e5e5).
- **Internal Padding:** 1rem.

### Inputs / Fields
- **Ricerca pubblica:** campo grande (1.25rem), bordo rgba(0,0,0,.2), radius 0.25rem, icona della lente a sinistra che diventa uno spinner mentre l'indice carica.
- **Backend:** a tutta larghezza, bordo #bbb, radius 4px, fondo bianco, padding 0.5rem, etichetta in grassetto sopra.
- **Focus:** quello del browser.

### Navigation
- **Header:** bianco con bordo inferiore #e0e0e0, logo ProgettoDighe 34px con il nome in semi-grassetto, voci del menu del sito; sul telefono un hamburger apre il menu.
- **Briciole:** blocchi pieni calcestruzzo, testo bianco grassetto 1rem in maiuscolo spaziato, una riga che va a capo.
- **Paginazione:** due pulsanti primari («‹ Precedenti», «Successive ›») con «Pagina N di M» in mezzo.
- **Backend:** barra bianca fissa in alto con marchio, ricerca, utente e widget delle operazioni.

### Visore della Foto
L'immagine a schermo con radius 1rem e ombra «Foglio». Ai lati, frecce semitrasparenti (rgba(0,0,0,.35), più scure al passaggio) alte 64px, attaccate al bordo. Si naviga anche con le frecce della tastiera e con lo scorrimento del dito.

### Griglia dei metadati
Righe a tre colonne: icona, etichetta in maiuscolo spaziato, valore in grassetto medio. I valori che sono link (Entità, Contributore, Licenza) hanno un leggero padding e si evidenziano in giallo cantiere al passaggio, con un sollevamento di 2px.

## Do's and Don'ts

### Do:
- **Do** usare le classi Tachyons già presenti prima di scrivere CSS nuovo; le regole proprie vanno in `backend/modelli/stile.css` (pubblico) o `backend/modelli/backend/backend.css` (Backend).
- **Do** mettere in maiuscolo spaziato (0.1em) etichette, intestazioni di sezione, briciole e pulsanti.
- **Do** dare alle miniature un segnaposto grigio chiaro (#eee) e un rapporto fisso (1:1 o 4:3).
- **Do** usare il calcestruzzo (#333) per ogni azione principale, sia nella Gallery sia nel Backend.
- **Do** progettare il Backend prima per il telefono: campi a tutta larghezza, azioni fisse in basso.

### Don't:
- **Don't** modificare i file copiati dal sito in `backend/modelli/sito/`: si ricopiano dal Sito principale (`TestCopieSito`).
- **Don't** usare il giallo cantiere come fondo di sezioni o titoli: segnala solo l'interazione.
- **Don't** mostrare nella Gallery pubblica i colori di stato del Backend, in particolare l'arancio Interno.
- **Don't** aggiungere web font o CDN: il font è quello di sistema, e le librerie stanno nel repo.
- **Don't** far sembrare il Backend una pagina del sito: niente header e piede del Sito principale.
