# Gerbillo

Gallery fotografica della community di ProgettoDighe, che sostituisce Coppermine. Le foto documentano dighe, centrali e altre opere descritte nel sito principale.

## Language

### Foto e media

**Foto**:
L'unità della gallery: un file pubblicato con i suoi metadati, identificato da un pid. È Foto qualunque sia il Formato: anche una scansione o un PDF.
_Avoid_: immagine, file, picture, elemento, documento

**Formato**:
Il tipo di file di una Foto (immagine, PDF, …). Determina quali Varianti si possono generare.
_Avoid_: tipo, estensione

**pid**:
Identificativo numerico stabile di una Foto, ereditato da Coppermine e proseguito per le foto nuove. Non cambia mai, qualunque sia la classificazione.
_Avoid_: id, codice

**Originale**:
Il file da cui derivano tutte le Varianti: quello caricato, a piena risoluzione, con gli EXIF ridotti a un elenco chiuso ed eventualmente ricompresso. È conservato su S3 e non è pubblico (ADR 0015).
_Avoid_: master, sorgente

**Variante**:
Una versione derivata dell'Originale (miniatura, versione a schermo, ecc.), servita a un URL legato al pid. La più grande (2048 px sul lato lungo) è quella che si scarica dalla pagina della Foto.
_Avoid_: thumb, normal (termini Coppermine)

**Licenza**:
La licenza sotto cui ogni Foto è pubblicata, mostrata nei suoi metadati. Se chi carica non ne sceglie un'altra da un elenco chiuso, è quella del Sito principale (CC BY-SA 4.0, ADR 0010).

### Classificazione

**Entità**:
Un oggetto che ha (o avrà) una pagina nel sito principale con uno Slug stabile: una diga, una centrale, un'opera di presa con pagina, un reportage. Una Foto è collegata alle Entità che ritrae tramite il campo `slug` (ADR 0013).
_Avoid_: oggetto, nodo, album

**Impianto**:
Un insieme di più Entità che funzionano insieme (es. diga, centrale e opere di presa di uno stesso sistema). Non è un'Entità: non ha una pagina propria nel Sito principale, dove oggi è identificato in modo non standardizzato (tag o id di una mappa). Ogni Entità appartiene al massimo a un Impianto. Una Foto che ritrae l'insieme (schemi, documenti) può essere collegata direttamente all'Impianto.
_Avoid_: complesso, sistema

**Slug**:
Il percorso stabile `sezione/nome` con cui il sito principale identifica una pagina (es. `dighe/agaro`, `centrali/goglio-baceno-vb`).

**Attributo**:
Una proprietà di una Foto che non è un collegamento a un'Entità: Tag, Categoria, Datazione, Autore, Fonte, Formato, Punto geografico.
_Avoid_: tag, campo

**Tag**:
Un'etichetta libera ma da vocabolario controllato, con gli stessi valori dei tag del Sito principale quando esistono (campo `tags`).

**Categoria**:
Il tipo o sottotipo del soggetto di una Foto, con valori a percorso (es. `tecnica/turbine`, campo `categories`).
_Avoid_: tema

**Punto geografico**:
Il punto geografico associato alla Foto (`geolocation.coords`), come nel sito lo è per l'articolo. Viene dal GPS dell'EXIF se chi carica lo consente, oppure si inserisce o corregge a mano, e il valore a mano vince. Se manca si eredita dall'Entità.

**Posizione nascosta**:
La scelta di chi carica di non pubblicare il GPS di una Foto: il GPS si toglie dall'Originale prima di salvarlo, e il Punto geografico si eredita dall'Entità. Riguarda solo la posizione: data, Autore ed EXIF utili restano.
_Avoid_: anonimizzazione

**Datazione**:
Il momento in cui la Foto è stata scattata, con precisione esplicita (giorno, mese, anno, decennio o intervallo), in EDTF. Distinta dalla Data di scansione e dalla Data di caricamento.
_Avoid_: data, periodo

**Data di scansione**:
Il momento in cui una foto analogica è stata digitalizzata.

**Data di caricamento**:
Il momento in cui la Foto è entrata nella gallery (campo `date`, come nel sito).

**Autore**:
Chi ha scattato la Foto: una persona o un ente, non necessariamente un utente del forum.
_Avoid_: uploader (chi carica può non essere l'Autore: vedi Contributore)

**Contributore**:
L'utente del forum che carica una Foto nella gallery. Può coincidere con l'Autore oppure no.
_Avoid_: uploader, utente

**Fonte**:
Da dove proviene la Foto quando non è opera di chi la carica (archivio, pubblicazione, ente).

**Vista**:
Un modo generato di navigare le Foto: per albero (regione → diga → Categoria), per periodo, per mappa, per Tag o Categoria, per Contributore. Nessuna Vista è la struttura reale dei dati.
_Avoid_: gerarchia, album

### Sistemi

**Sito principale**:
Il sito Hugo di ProgettoDighe, fonte di verità delle Entità.
_Avoid_: sito Hugo, sito statico

**Gallery pubblica**:
Il sito statico che mostra le Foto ai visitatori, generato dal binario Go (ADR 0012).
_Avoid_: frontend, sito gallery

**Backend**:
Il servizio per upload, moderazione e strumenti editoriali. Non serve pagine al pubblico.
_Avoid_: admin, server

**Indice delle Entità**:
Il file prodotto dal Sito principale che elenca le Entità per la gallery.

**Indice delle Foto**:
Il file prodotto dalla gallery che elenca le Foto per Entità, usato dal Sito principale.

**Indice di ricerca**:
Il file (`cerca.json`) che la pagina di ricerca della Gallery pubblica (`/cerca/`) scarica e indicizza nel browser: Entità, Contributori e Foto (ADR 0014).
_Avoid_: indice (da solo), Indice delle Foto (che è per il Sito principale)

**Repository dei contenuti**:
Il repository git solo locale sul server, separato dal codice, che contiene i metadati pubblicati delle Foto (ADR 0004).

### Flusso editoriale

**Proposta**:
Una Foto caricata e non ancora pubblicata. Può essere in coda, in bozza o rifiutata.
_Avoid_: upload, submission

**Moderatore**:
Un amministratore o moderatore del forum, autorizzato ad approvare, rifiutare e organizzare le Foto.
_Avoid_: admin, revisore

**Pubblicazione**:
Il passaggio di una Proposta a Foto: commit dei metadati nel Repository dei contenuti e rigenerazione della Gallery pubblica.

**Copia per il forum**:
Il BBCode generato per una Foto: miniatura cliccabile verso la pagina della Foto.

### Migrazione

**Coppermine**:
La gallery attuale, fonte primaria delle Foto finché resta attiva.

**Album** / **Categoria Coppermine**:
Contenitori di Coppermine. Esistono solo come input della Mappatura, non nel modello di Gerbillo. Da non confondere con la Categoria di Gerbillo.

**Album personale**:
Un Album di Coppermine nelle *User galleries*, legato a un Contributore e non a un'Entità. Come gli altri Album sparisce come contenitore ed entra nella Mappatura; le Foto di ogni Contributore si trovano nella Vista per Contributore (#2).

**Importazione**:
L'esecuzione, ripetibile, che porta le Foto da Coppermine a Gerbillo. Ogni esecuzione rigenera da zero i dati importati, che non contengono mai lavoro editoriale: quello sta nel Livello di arricchimento, e la Foto pubblicata nasce dalla loro unione.
_Avoid_: sync, esportazione, migrazione (la Migrazione è il processo complessivo)

**Mappatura**:
La corrispondenza tra un Album (o una Categoria Coppermine) e gli Slug, i Tag e le Categorie di Gerbillo.

**Livello di arricchimento**:
Il lavoro editoriale fatto in Gerbillo (Mappature, correzioni, metadati aggiunti), applicato sopra ogni Importazione.
_Avoid_: overlay, patch

**Switch**:
Il momento in cui Coppermine viene spento e Gerbillo diventa la fonte primaria.

**Bonifica**:
La riscrittura dei post del forum che puntano a URL di Coppermine.
