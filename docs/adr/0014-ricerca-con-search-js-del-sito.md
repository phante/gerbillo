# Ricerca della Gallery con `search.js` del Sito principale e un indice JSON unico

Deciso nel grilling del 24/10/2026, per #26. Precisa la scelta lasciata aperta da ADR 0012 («Pagefind o un indice JSON»).

La ricerca della Gallery pubblica è quella del Sito principale: `search.js` (FlexSearch incluso) copiato byte per byte come gli altri asset (#25), con `window.searchIndexPath` che punta all'Indice di ricerca, `cerca.json`, generato da `gerbillo build` nello stesso formato di `/index.json` del sito. Il campo sta in home, sopra la Bussola, e compare solo con JavaScript.

L'indice contiene le pagine delle Entità (pillole Dighe, Centrali, Altre pagine), i Contributori e ogni Foto. Per la Foto: `title` (con il ripiego sull'Entità), in `meta` nomi delle Entità, Autori e Contributore, in `tags` Tag e Categorie, in `content` descrizione, luogo e Datazione. Lo snippet è `Entità · Datazione · Autore — descrizione…`.

Il motivo è la coerenza: stessa ricerca, stesso aspetto, nessun JavaScript nuovo e nessuna dipendenza nella build. Misure su 25.617 Foto reali: indice di circa 6 MB (600 KB con gzip), costruito nel browser in 1,5 s e 114 MB su una CPU da server; poi ogni ricerca richiede pochi millisecondi. Le pagine delle Entità escono prima delle Foto.

## Considered Options

- **Pagefind**: carica solo i pezzi d'indice che servono, ma è un secondo binario nella build (contro ADR 0012) e ha un'interfaccia sua.
- **Indice a pezzi generato in Go e JavaScript scritto a mano**: leggero anche su telefono, ma è codice da mantenere. È il ripiego se l'attesa alla prima ricerca dall'iPhone risulta fastidiosa.
- **`search.js` modificato** (miniature, più di 10 risultati): perderebbe l'allineamento con il sito verificato da `TestCopieSito`.

## Consequences

- Valgono i limiti di `search.js`: 10 risultati visibili, nessuna miniatura, conteggi delle pillole calcolati sui primi 30 risultati per campo. La ricerca porta alle pagine; per sfogliare restano le pagine delle Entità.
- La prima ricerca scarica e indicizza tutto: qualche secondo su telefono.
- Rimandati: ricerca dentro la pagina della diga, pagine per Autore, Tag e Categoria.
