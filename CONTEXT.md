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
Il file caricato così com'è, da cui derivano tutte le Varianti.
_Avoid_: master, sorgente

**Variante**:
Una versione derivata dell'Originale (miniatura, versione a schermo, ecc.), servita a un URL legato al pid.
_Avoid_: thumb, normal (termini Coppermine)

**Licenza**:
La licenza sotto cui ogni Foto è pubblicata: unica, obbligatoria e uguale a quella del sito principale.

### Classificazione

**Entità**:
Un oggetto che ha (o avrà) una pagina nel sito principale con uno slug stabile: una diga, una centrale, un'opera di presa con pagina. Una Foto è di norma collegata a una o più Entità oppure a un Impianto, raramente a entrambi.
_Avoid_: oggetto, nodo, album

**Impianto**:
Un insieme di più Entità che funzionano insieme (es. diga, centrale e opere di presa di uno stesso sistema). Non è un'Entità: non ha una pagina propria nel Sito principale, dove oggi è identificato in modo non standardizzato (tag o id di una mappa). Ogni Entità appartiene al massimo a un Impianto. Una Foto che ritrae l'insieme (schemi, documenti) può essere collegata direttamente all'Impianto.
_Avoid_: complesso, sistema

**Slug**:
Il nome stabile con cui il sito principale identifica un'Entità (es. `agaro`).

**Attributo**:
Una proprietà di una Foto che non è un collegamento a un'Entità: Tema, Datazione, Autore, Fonte, Formato.
_Avoid_: tag, campo

**Tema**:
Un Attributo che classifica il soggetto di una Foto senza essere un'Entità (es. Tecnica, Meeting, opere di presa senza pagina).
_Avoid_: categoria, tag

**Datazione**:
Il momento in cui la Foto è stata scattata, con precisione esplicita (giorno, mese, anno, decennio o intervallo). Distinta dalla Data di scansione e dalla Data di caricamento.
_Avoid_: data, periodo

**Data di scansione**:
Il momento in cui una foto analogica è stata digitalizzata.

**Data di caricamento**:
Il momento in cui la Foto è entrata nella gallery.

**Autore**:
Chi ha scattato la Foto.
_Avoid_: uploader (chi carica può non essere l'Autore: vedi Contributore)

**Contributore**:
L'utente del forum che carica una Foto nella gallery. Può coincidere con l'Autore oppure no.
_Avoid_: uploader, utente

**Fonte**:
Da dove proviene la Foto quando non è opera di chi la carica (archivio, pubblicazione, ente).

**Vista**:
Un modo generato di navigare le Foto: per albero (regione → diga → tema), per periodo, per mappa, per Tema. Nessuna Vista è la struttura reale dei dati.
_Avoid_: gerarchia, album

### Sistemi

**Sito principale**:
Il sito Hugo di ProgettoDighe, fonte di verità delle Entità.
_Avoid_: sito Hugo, sito statico

**Gallery pubblica**:
Il sito statico che mostra le Foto ai visitatori.
_Avoid_: frontend, sito gallery

**Backend**:
Il servizio per upload, moderazione e strumenti editoriali. Non serve pagine al pubblico.
_Avoid_: admin, server

**Indice delle Entità**:
Il file prodotto dal Sito principale che elenca le Entità per la gallery.

**Indice delle Foto**:
Il file prodotto dalla gallery che elenca le Foto per Entità, usato dal Sito principale.

**Repository dei contenuti**:
Il repository git, separato dal codice, che contiene i metadati pubblicati delle Foto.

### Flusso editoriale

**Proposta**:
Una Foto caricata e non ancora pubblicata. Può essere in coda, in bozza o rifiutata.
_Avoid_: upload, submission

**Moderatore**:
Un amministratore o moderatore del forum, autorizzato ad approvare, rifiutare e organizzare le Foto.
_Avoid_: admin, revisore

**Pubblicazione**:
Il passaggio di una Proposta a Foto, con scrittura dei metadati nel Repository dei contenuti.

**Copia per il forum**:
Il BBCode generato per una Foto: miniatura cliccabile verso la pagina della Foto.

### Migrazione

**Coppermine**:
La gallery attuale, fonte primaria delle Foto finché resta attiva.

**Album** / **Categoria**:
Contenitori di Coppermine. Esistono solo come input della Mappatura, non nel modello di Gerbillo.

**Album personale**:
Un Album di Coppermine nelle *User galleries*, legato a un Contributore e non a un'Entità. Il suo destino nel modello è da decidere guardando i dati di G1.

**Importazione**:
L'esecuzione, ripetibile, che porta le Foto da Coppermine a Gerbillo.
_Avoid_: sync, migrazione (la Migrazione è il processo complessivo)

**Mappatura**:
La corrispondenza tra un Album di Coppermine e le Entità o i Temi di Gerbillo.

**Livello di arricchimento**:
Il lavoro editoriale fatto in Gerbillo (Mappature, correzioni, metadati aggiunti), applicato sopra ogni Importazione.
_Avoid_: overlay, patch

**Switch**:
Il momento in cui Coppermine viene spento e Gerbillo diventa la fonte primaria.

**Bonifica**:
La riscrittura dei post del forum che puntano a URL di Coppermine.
