# Il pid è l'identificativo permanente e la base degli URL

Ogni Foto è identificata da un pid, che non cambia mai. Gli URL dei media dipendono solo dal pid, non dalla classificazione: `/gallery/f/<pid>/<variante>.webp`, e per un PDF la Variante documento `/gallery/f/<pid>/documento.pdf` (ADR 0015, #90). La configurazione di nginx per `/gallery/f/` deve servire entrambi. Riclassificare una Foto non rompe nessun link.

Rivisto il 28/11/2026: il pid ha due forme.

- **Le Foto di Coppermine**, sia quelle importate sia quelle che arrivano con la Sincronizzazione, tengono il pid di Coppermine: solo cifre, al massimo 7.
- **Le Foto caricate in Gerbillo** hanno un ULID in minuscolo, per esempio `01jdq3x8k2m4n5p6r7s8t9v0wx`: 26 caratteri in base32 Crockford, 48 bit di millisecondi e 80 bit casuali da `crypto/rand`, generato senza dipendenze.

Le due forme non si confondono mai, quindi Coppermine e Gerbillo possono assegnare pid insieme durante la convivenza senza collisioni. Allo Switch, ogni URL di Coppermine (pagine e file `albums/…`, comprese le varianti `thumb_` e `normal_`) si mappa con un redirect 301 deterministico, perché contiene un pid numerico che in Gerbillo è lo stesso. Gli ULID non hanno nulla da redirigere.

Il pid è **opaco**: non dice nulla sull'ordine delle Foto. Ogni ordinamento usa un campo esplicito (data di caricamento, Datazione, titolo), e il pid solo per spareggiare.

## Considered Options

- **Pid numerico per tutti, con le Foto nuove da una soglia alta**: scartato il 28/11/2026. Evita le collisioni, ma tiene l'idea che il pid indichi l'ordine, e ordina male le cifre in `ls`, git e negli strumenti esterni.
- **UUID o ULID per tutte le Foto, con il pid di Coppermine come secondo campo**: scartato. Due identificativi per Foto per sempre, solo per i redirect.
- **UUIDv7**: scartato per l'ULID. Sono 36 caratteri contro 26, negli URL e nel BBCode del forum, e la libreria standard di Go non lo genera.

## Consequences

- Nel codice il pid diventa una stringa. Dove arriva dall'esterno (URL, moduli) si controlla la forma: solo cifre, oppure un ULID valido. Va fatto prima dell'Avvio e prima del caricamento dal Backend (#29): dopo, cambiarlo vorrebbe dire migrare i dati veri.
- L'Importazione scrive il percorso Coppermine (`filepath` + `filename`) nella voce `coppermine` di `original` (ADR 0021), senza campi `coppermine.filepath`/`filename` separati (rivisto il 28/11/2026). Allo Switch, **prima** di togliere le voci `coppermine`, un comando genera una volta sola dai YAML la mappa nginx dei redirect dei file `albums/…`. Da lì la mappa è congelata: gli URL di Coppermine non cambiano più, quindi non c'è niente da tenere allineato. Se l'ordine si inverte, i percorsi restano nella storia git e nel DB di Coppermine.
- Gli Originali restano fuori da git. Le Varianti (WebP) sono generate da un comando idempotente che crea solo ciò che manca. Dal 15/10/2026 l'encoder è `github.com/gen2brain/webp` (libwebp compilato in WASM e tradotto in Go puro, senza cgo), al posto di libvips. AVIF è rimandato finché WebP non si dimostra insufficiente.
- Nessuna garanzia per i siti esterni che fanno hotlink diretto.
