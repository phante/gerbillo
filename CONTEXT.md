# Gerbillo

Gallery fotografica della community di ProgettoDighe, che sostituisce Coppermine. Le foto documentano dighe, centrali, altre opere ed eventi. Gallery e Sito principale lavorano in sinergia, senza vincoli (ADR 0019).

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
Un soggetto con un nome proprio, che ha una pagina nella Gallery pubblica: un'opera, un evento, un reportage. Il test è «si può dire *la X*?»: «le turbine Pelton» non è un'Entità, ma una Categoria. Una Foto è collegata alle Entità che ritrae tramite il campo `slug` (ADR 0013). Chi carica sceglie l'Entità per nome, tipo e comune («Cosa ritrae?»), mai per Slug. Le Entità vengono da due fonti unite al build: i file di `entita/` e l'Indice delle Entità (ADR 0019).
_Avoid_: oggetto, nodo, album

**Entità senza pagina**:
Un'Entità che ha una pagina nella Gallery ma non sul Sito principale. Sta in `entita/<sezione>/<nome>.yaml` e le Foto la indicano con `slug`, come tutte. La crea chi carica quando non trova l'opera, e il sistema genera lo Slug. Quando il sito pubblica la pagina con lo stesso Slug smette di esserlo da sola; con uno Slug diverso, un Moderatore la collega e quello vecchio diventa un Alias (ADR 0018). È un'Informazione interna.
_Avoid_: Entità da censire, opera da censire, segnalazione, Entità provvisoria

**Alias**:
Un vecchio Slug di un'Entità, elencato in `aliases`: il build scrive al suo posto una pagina di rimando all'URL attuale. Nasce da un collegamento, una rinomina o un'unione di doppioni, e si vede nella scheda del Backend (ADR 0018).
_Avoid_: redirect, `diventata`

**Informazione interna**:
Ciò che serve solo a Moderatori e amministratori: resta nel Backend, e il build pubblico non lo scrive mai in `dist/`. I tag interni ne sono un esempio. Un'informazione nuova nasce interna, salvo decisione contraria (ADR 0019).
_Avoid_: informazione editoriale

**Impianto**:
Un insieme di più Entità che funzionano insieme (es. diga, centrale e opere di presa di uno stesso sistema). È un concetto del Sito principale, che in futuro lo ricaverà dalle schede di dighe e centrali (`dighe:`, `centrali:`); oggi vi è identificato in modo non standardizzato (tag o id di una mappa, che non coincide sempre con un Impianto). Nella gallery non è modellato: un insieme di Entità si esprime con un Tag, senza prefisso (#3).
_Avoid_: complesso, sistema, campo `impianto`

**Slug**:
Il percorso `sezione/nome` che identifica un'Entità (es. `dighe/agaro`, `centrali/goglio-baceno-vb`), lo stesso formato del Sito principale: se il sito ha lo stesso Slug, è la stessa Entità. È un riferimento tecnico: lo scrive il sistema, e chi carica non lo vede.

**Attributo**:
Una proprietà di una Foto che non è un collegamento a un'Entità: Tag, Categoria, Datazione, Autore, Fonte, Formato, Punto geografico.
_Avoid_: tag, campo

**Tag**:
Un'etichetta libera ma da vocabolario controllato, con gli stessi valori dei tag del Sito principale quando esistono (campo `tags`).

**Categoria**:
Il genere del soggetto di una Foto, con valori a percorso (es. `tecnica/turbine`, campo `categories`). Dice di che genere è, mentre l'Entità dice chi è. Una Foto senza Entità e senza Categoria è «Da classificare» nel Backend e sta in «Altre foto» nella Gallery pubblica.
_Avoid_: tema

**Punto geografico**:
Il punto geografico associato alla Foto (`geolocation.coords`), come nel sito lo è per l'articolo. Viene dal GPS dell'EXIF se chi carica lo consente, oppure si inserisce o corregge a mano, e il valore a mano vince. Se la Foto non ne ha uno, vale quello della sua prima Entità: non si copia nella Foto, si calcola quando serve (ADR 0013). Senza Entità, o se l'Entità non ha un punto, la Foto resta senza.

**Posizione nascosta**:
Il rifiuto del GPS da parte di chi carica, prima del caricamento: il GPS si toglie dall'Originale prima di salvarlo. Non vuol dire restare senza posizione: chi carica può indicare un Punto geografico a mano, altrimenti vale quello dell'Entità. Riguarda solo il GPS: data, Autore ed EXIF utili restano.
_Avoid_: anonimizzazione

**Consenso al GPS**:
L'autorizzazione di chi carica a usare, mantenere e pubblicare il GPS di una Foto, chiesta prima di ogni caricamento con GPS e registrata come Informazione interna. Si ritira con una mail, non con una funzione del Backend (ADR 0015).

**Datazione**:
Il momento in cui la Foto è stata scattata, con precisione esplicita (giorno, mese, anno, decennio o intervallo), in EDTF. Distinta dalla Data di scansione e dalla Data di caricamento. Nelle Foto importate viene dall'EXIF (#18); da `user3` di Coppermine, con regole fisse, solo per le candidate: le Foto senza EXIF con `user3` più di un anno prima della Data di caricamento, e le scansioni, con l'EXIF più di un anno dopo `user3`. Per le scansioni importate la Data di scansione non si conserva: l'EXIF resta solo grezzo (#39). Per le altre Foto `user3` si mostra come testo, se manca la Datazione.
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
Il sito Hugo di ProgettoDighe, separato dalla Gallery. Quando ha una pagina per un'Entità, ne fornisce link e scheda dati; altrimenti la Gallery funziona lo stesso.
_Avoid_: sito Hugo, sito statico

**Gallery pubblica**:
Il sito statico che mostra le Foto ai visitatori, generato dal binario Go (ADR 0012).
_Avoid_: frontend, sito gallery

**Backend**:
Il servizio per upload, moderazione e strumenti editoriali. Non serve pagine al pubblico.
_Avoid_: admin, server

**Indice delle Entità**:
Il file prodotto dal Sito principale che elenca le sue pagine per la gallery. È una delle due fonti delle Entità (ADR 0019).

**Indice delle Foto**:
Il file prodotto dalla gallery che elenca le Foto per Slug, usato dal Sito principale.

**Indice di ricerca**:
Il file (`cerca.json`) che la pagina di ricerca della Gallery pubblica (`/cerca/`) scarica e indicizza nel browser: Entità, Contributori e Foto (ADR 0014).
_Avoid_: indice (da solo), Indice delle Foto (che è per il Sito principale)

**Repository dei contenuti**:
Il repository git solo locale sul server, separato dal codice, che contiene i metadati pubblicati delle Foto (ADR 0004).

### Flusso editoriale

**Proposta**:
Una Foto caricata e non ancora pubblicata. Può essere in coda, in bozza o rifiutata.
_Avoid_: upload, submission

**Correzione**:
Una modifica proposta da un utente del forum a una Foto già pubblicata: valori nuovi per uno o più campi, con una nota facoltativa. Una Correzione con la sola nota è una segnalazione. Può essere aperta, approvata (anche in parte) o rifiutata.
_Avoid_: modifica, suggerimento, segnalazione

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
La corrispondenza tra un Album (o una Categoria Coppermine) e gli Slug, i Tag e le Categorie di Gerbillo. È un dato derivato, calcolato a ogni build: per ogni Album vale la riga manuale, altrimenti l'abbinamento automatico sul nome, altrimenti la Mappatura della categoria. Le righe manuali stanno in `arricchimento/mappature.yaml`, scritto solo a mano: una riga sostituisce in blocco l'abbinamento automatico, una riga senza Slug vuol dire "nessuna Entità". Gli Album senza nessuna delle tre hanno il tag interno `da-collegare`; quelli da fare e gli ambigui sono nel rapporto dell'Importazione (#38).

**Livello di arricchimento**:
Il lavoro editoriale fatto in Gerbillo (Mappature, correzioni, metadati aggiunti), applicato sopra ogni Importazione.
_Avoid_: overlay, patch

**Switch**:
Il momento in cui Coppermine viene spento e Gerbillo diventa la fonte primaria.

**Bonifica**:
La riscrittura dei post del forum che puntano a URL di Coppermine.
