# Il pid di Coppermine è l'identificativo permanente e la base degli URL

Ogni Foto è identificata dal `pid` di Coppermine, e le foto nuove proseguono la numerazione. Gli URL dei media dipendono solo dal pid, non dalla classificazione: `/gallery/f/<pid>/<variante>.webp`. In questo modo, allo Switch, ogni URL di Coppermine (pagine e file `albums/…`, comprese le varianti `thumb_` e `normal_`) si mappa con un redirect 301 deterministico. Riclassificare una Foto non rompe nessun link.

## Consequences

- L'Importazione conserva nel YAML di ogni Foto il percorso Coppermine (`filepath` e `filename`). I redirect dei file e la ricerca degli Originali si ricavano da lì, senza una tabella separata da tenere allineata. Questo dato di provenienza resta anche dopo lo Switch.
- Gli Originali restano fuori da git. Le Varianti (WebP) sono generate da un comando idempotente che crea solo ciò che manca. Dal 15/10/2026 l'encoder è `github.com/gen2brain/webp` (libwebp compilato in WASM e tradotto in Go puro, senza cgo), al posto di libvips. AVIF è rimandato finché WebP non si dimostra insufficiente.
- Nessuna garanzia per i siti esterni che fanno hotlink diretto.
