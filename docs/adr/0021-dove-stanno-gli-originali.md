# Il YAML della Foto dice dove sta l'Originale

Deciso il 28/11/2026, nel grilling sulla gestione degli Originali. Completa ADR 0015, che dice cosa è l'Originale e perché sta fuori dal server; qui si decide come Gerbillo sa dove trovarlo.

Gli Originali stanno in più posti, che cambiano nel tempo: `albums/` di Coppermine durante la convivenza, una cartella locale finché un caricamento non è arrivato fuori dal server, S3 e B2 a regime, domani forse altro. Gerbillo non deve ricavare il posto da una regola nel codice: ogni Foto lo dichiara.

## Il campo `original`

```yaml
original:
  - store: coppermine
    path: userpics/10025/2005-monte_diga.jpg
    sha256: 4be1…
  - store: s3
    path: 2005/06/10025.jpg
    sha256: 9f2c…
  - store: b2
    path: 2005/06/10025.jpg
    sha256: 9f2c…
```

- **Una voce per ogni copia che esiste**, con lo store, il percorso relativo e l'impronta di quel file.
- **Si legge dall'alto.** Gerbillo prova la prima voce; se lo store non risponde o l'impronta non coincide, passa alla successiva. L'ordine è quello scritto: nessuna configurazione dice come leggere il dato.
- **Nel YAML solo nomi logici.** Per ogni store, l'ambiente dice il tipo e dove sta (cartella, bucket, credenziali), come in ADR 0020. Cambiare bucket o fornitore è una variabile, non 25.000 YAML.
- **Il campo non si pubblica** nella Gallery.

## Store

Uno store è un posto dove sta una copia dell'Originale. Ce ne sono di due tipi:

- **Gestiti** (`gerbillo`, `s3`, `b2`): Gerbillo ci scrive. Le copie in store gestiti sono lo stesso file, quindi hanno la stessa impronta. Contano per le due copie fuori dal server di ADR 0015.
- **Sorgenti** (`coppermine`, domani `forum`): Gerbillo le legge e non ci scrive né cancella mai. Il file ha gli EXIF di chi l'ha caricato, quindi un'impronta diversa dalle copie gestite è normale. Non contano per le due copie.

`gerbillo` è la cartella locale dell'istanza: tiene l'Originale appena caricato finché gli store remoti non l'hanno confermato. Sta fuori dalle Varianti servite da nginx, perché l'Originale non è pubblico, e fuori dal Repository dei contenuti.

Importare da una fonte nuova, per esempio le immagini del forum, vuol dire scriverne il lettore: Varianti, Riorganizzazione e copie remote funzionano già.

## Ciclo di una Foto caricata in Gerbillo

1. Il caricamento scrive l'Originale, con gli EXIF filtrati, in `gerbillo` a `AAAA/MM/<pid>.<ext>` (mese del caricamento) e ne calcola lo `sha256`. Il nome del file caricato si butta: può contenere dati personali.
2. Gerbillo lo copia su ogni store remoto gestito, con lo stesso percorso. La voce si aggiunge in fondo, con un commit, solo quando lo store conferma il checksum SHA-256 (le API di S3 e B2 lo verificano al caricamento).
3. Quando tutti gli store remoti configurati hanno la copia, la Foto entra in «Copie locali da liberare».
4. Un admin approva la cancellazione della copia locale, a blocchi dalla pagina admin degli Originali o per la singola Foto dalla sua scheda. Il file si cancella e la voce `gerbillo` esce, con un commit.

Cosa succede se uno store non risponde (tentativi, avvisi) è in #30.

## Foto di Coppermine

- L'Importazione e la Sincronizzazione scrivono una sola voce `coppermine`, con `filepath` + `filename` di Coppermine e l'impronta del file in `albums/`. Non esistono campi `coppermine.filepath` e `coppermine.filename` separati.
- La **Riorganizzazione** porta una Foto allo schema comune: scrive l'Originale con gli EXIF filtrati, senza ricomprimere, a `AAAA/MM/<pid>.<ext>` (mese del caricamento in Coppermine), lo copia sugli store gestiti e aggiunge le voci. È uno strumento generale, per qualsiasi Foto fuori schema.
- Durante la convivenza la Riorganizzazione si può già fare: la voce `coppermine` resta, perché Coppermine usa ancora quei file. Non occupa spazio sul server, perché le copie vanno sugli store remoti.
- Allo Switch, in quest'ordine: si genera dalle voci `coppermine` la mappa nginx dei redirect di `albums/…` (ADR 0005), che da lì resta congelata; poi si riorganizzano le Foto che mancano e si toglie la voce `coppermine` da tutte, in un commit.

## Verifica

L'impronta di una voce si confronta con il file che indica, non con le altre voci.

- **A ogni lettura** (Varianti, rigenerazione di #138) Gerbillo ricalcola l'hash dei byte che ha già in mano. Se non coincide scarta la voce, passa alla successiva e avvisa su Telegram.
- **Una verifica periodica** nel Backend, con intervallo nell'ambiente, passa tutte le voci: per S3 e B2 legge il checksum salvato (HeadObject, senza scaricare), per `gerbillo` e `coppermine` ricalcola l'hash dal disco.
- **La Sincronizzazione** segnala in «Da sistemare» una Foto di Coppermine il cui file è cambiato.
- **Due copie gestite con impronte diverse** sono un errore, visibile dal YAML stesso.

Nessuna verifica corregge o cancella: avvisa e basta.

Gli esiti stanno nella cache dell'istanza, fuori dal Repository dei contenuti: un commit per ogni controllo sarebbe rumore. Tutto quello che sta nella cache si può cancellare in ogni momento; il Backend lo ricostruisce.

Nella scheda della Foto, solo per l'admin, il riquadro **Originale** mostra una riga per voce: store, tipo, percorso, impronta e stato (verificato con la data, diverso dalle altre copie gestite, file cambiato, non raggiungibile, mai verificato). La differenza di impronta di una sorgente non si segnala. Un pulsante **Verifica ora** ricontrolla le voci della Foto. Le Foto con un problema compaiono in «Da sistemare».

## Abbandonare uno store

Un comando copia gli Originali dove serve e toglie la voce dello store da tutti i YAML, in un commit. Si rifiuta se una Foto resterebbe con meno di due copie remote gestite.

## Considered Options

- **Percorso ricavato da `coppermine.filepath` e `filename`, o da una regola nel codice**: scartato. Vale solo per gli Originali di Coppermine, e la regola cambierebbe a ogni nuovo store.
- **Un percorso unico e l'elenco degli store che hanno la copia**: scartato. Impone lo stesso percorso ovunque, e la Riorganizzazione deve poter rinominare.
- **Mappa store → percorso, con l'ordine di lettura nell'ambiente**: scartato. Una mappa YAML non ha ordine, e una variabile in più per spiegare come leggere un dato è configurazione oscura.
- **Un solo `sha256` per Foto**: scartato. Il file di una sorgente ha byte diversi dalle copie gestite, e diventerebbe un'eccezione.
- **Esiti delle verifiche nel YAML**: scartato. Un commit per ogni controllo.
- **Esiti solo in memoria**: scartato. Si perdono a ogni riavvio, e la verifica ripartirebbe da capo.
- **Copia locale cancellata in automatico dopo la conferma remota**: scartato. Fino a lì è l'unica copia sotto il controllo diretto dell'admin: la cancellazione la approva lui.

## Consequences

- L'Importazione legge tutti i file di `albums/` per calcolarne l'impronta. Si fa sul Mac, prima dell'Avvio.
- Il comando `varianti` e i link della Gallery a `albums/` leggono `original` invece di `coppermine.filepath`/`filename`. Va fatto prima dell'Avvio, come il pid stringa (ADR 0005).
- Ogni copia confermata è un commit nel Repository dei contenuti: con pochi caricamenti al mese è rumore accettabile.
