# Il pid di Coppermine è l'identificativo permanente e la base degli URL

Ogni Foto è identificata dal `pid` di Coppermine, e le foto nuove proseguono la numerazione. Gli URL dei media dipendono solo dal pid, non dalla classificazione: `/gallery/f/<pid>/<variante>.webp`. In questo modo, allo Switch, ogni URL di Coppermine (pagine e file `albums/…`, comprese le varianti `thumb_` e `normal_`) si mappa con un redirect 301 deterministico. Riclassificare una Foto non rompe nessun link.

## Consequences

- L'esportazione di G1 produce fin da subito la tabella percorso + file + variante → pid.
- Gli Originali restano fuori da git. Le Varianti (WebP/AVIF) sono generate con libvips da un comando idempotente che crea solo ciò che manca.
- Nessuna garanzia per i siti esterni che fanno hotlink diretto.
