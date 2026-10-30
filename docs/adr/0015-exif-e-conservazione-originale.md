# EXIF filtrati, Originale su S3 e GPS solo con consenso

Deciso il 24/10/2026, nel grilling di #22. Vale per le Foto caricate nel Backend e, allo Switch, per quelle importate da Coppermine.

Con il caricamento dal telefono arriveranno foto a piena risoluzione (3–15 MB l'una), e un Contributore può caricarne decine a escursione: con 6.000 Foto l'anno sarebbero circa 36 GB l'anno sul server. L'Originale quindi si riscrive comunque, e riscriverlo permette di tenere solo i metadati che servono.

- **Originale su S3, non pubblico.** A piena risoluzione, in un bucket dedicato nello stesso account AWS del backup, con versioning. È la fonte delle Varianti; le Varianti si generano al caricamento dal file ricevuto, e da S3 si rilegge solo per rigenerarle. Il file caricato così com'è non si conserva.
- **Sul server solo le Varianti.** La più grande, 2048 px sul lato lungo, è quella che si scarica dalla pagina della Foto. Nessun pulsante per scaricare l'Originale: il Sito principale prenderà le immagini per gli articoli con un meccanismo suo, nel processo editoriale.
- **Compressione.** Originale in JPEG con mozjpeg, qualità circa 85: abbastanza per reggere un'altra compressione quando se ne ricava qualcosa. Le Varianti restano WebP (ADR 0005), con una qualità da sito, circa 75–80. Un file che pesa già meno di quanto peserebbe ricompresso non si ricomprime: se ne filtrano solo gli EXIF, senza perdite. I valori esatti si fissano con una prova su un campione di foto da telefono.
- **EXIF tenuti, elenco chiuso:** data e ora di scatto con il fuso, Artist, Copyright, marca, modello, obiettivo, esposizione (tempo, diaframma, ISO, focale). Il GPS solo se chi carica lo consente. Si buttano maker notes (dove stanno i numeri di serie), miniature incorporate e cronologia XMP. L'orientamento si applica ai pixel.
- **GPS solo con consenso.** Se un file ha il GPS, prima del caricamento un avviso dice che è un dato personale, mostra dove punta e chiede l'autorizzazione a usarlo, mantenerlo e pubblicarlo. L'avviso compare a ogni caricamento con GPS, con la scelta proposta in base all'ultima. Vale per tutto il caricamento e si può cambiare per la singola Foto. Il consenso si registra come informazione interna, con data e testo.
  - **Con un sì** il GPS resta nell'Originale, e il Punto geografico parte da lì. Chi carica può correggerlo: si pubblica il valore corretto, e l'Originale non cambia.
  - **Con un no** è una Posizione nascosta: il GPS non arriva né su S3 né nei metadati. Chi carica può indicare un punto a mano; altrimenti vale quello dell'Entità, se ne ha una (ADR 0013).
  - **Senza GPS nel file** non c'è nessun avviso: il punto si indica a mano, o vale quello dell'Entità.
- **Artist e Copyright diventano l'Autore proposto**, che chi carica conferma o corregge.
- **Dal GPS si suggerisce l'Entità** più vicina, entro 5 km, mostrando le 2–3 più vicine se le distanze sono simili.
- **Ritiro del consenso con una mail.** Non c'è una funzione per l'utente nel Backend. L'avviso al caricamento e l'informativa hanno un link `mailto:` a info@progettodighe.it, con oggetto e pid già compilati. Si risponde entro un mese: un Moderatore toglie il punto pubblico, l'admin ripulisce l'Originale.
- **Strumento admin sugli Originali.** Una pagina del Backend, solo per l'admin, mostra i metadati dell'Originale su S3 e permette di modificarli o toglierli, riscrivendo il file senza ricomprimere. La versione vecchia non si cancella dal Backend: una regola del bucket (S3 lifecycle) fa scadere le versioni precedenti dopo 30 giorni.
- **Foto importate.** Allo Switch gli Originali di Coppermine vanno su S3 con gli EXIF filtrati senza ricomprimere. Il loro GPS (circa 90 Foto, tutte da telefono, vicino alla diga) diventa Punto geografico e si pubblica; si toglie su richiesta. Il link pubblico agli Originali su `albums/` sparisce allo Switch.

## Perché

- Le coordinate GPS, con la data, sono dati personali (GDPR, art. 4). L'art. 25(2) chiede che non diventino accessibili a chiunque senza un intervento dell'interessato: basta una scelta consapevole, non serve che il sistema decida per lui. Chi carica spesso non sa che il GPS c'è, quindi glielo si mostra.
- Artist e Copyright di solito non sono impostati: se ci sono li ha voluti l'autore, e servono ai crediti (la CC BY-SA chiede l'attribuzione).
- Numeri di serie, maker notes e cronologia dei programmi non servono e pesano; in Coppermine il seriale c'è in circa il 9% delle foto e collega fra loro quelle della stessa fotocamera.
- La data resta sempre, anche con la Posizione nascosta: serve a capire come cambia un luogo.
- Il GDPR non chiede automatismi per il ritiro, ma che sia facile quanto dare il consenso (art. 7(3)) e che si risponda entro un mese (art. 12(3)). Il link già compilato basta a questo, con una riga di HTML. Il consenso va anche saputo dimostrare (art. 7(1)), per questo si registra.
- Il Backend è esposto su internet: se non può cancellare versioni, un errore o un'intrusione non può distruggere gli Originali. Il prezzo è che un errore si recupera solo entro 30 giorni.
- Le richieste di accesso ai propri dati (GDPR, art. 15 e 20) saranno rare: le gestisce a mano un admin dal bucket.

## Considered Options

- **Anonimizzazione completa al caricamento, data compresa**: scartata. Toglie la Datazione e fa per l'autore un lavoro che non ha chiesto.
- **Nessun intervento, solo un avviso**: scartata. Il GPS finirebbe online all'insaputa di chi carica.
- **Conservare il file caricato così com'è, in privato**: scartata. Terrebbe proprio il GPS che chi carica ha chiesto di togliere.
- **Originale a 4000 px sul server**: scartato. Con molti caricamenti dal telefono il server si riempie; la risoluzione piena sta meglio su S3.
- **Coordinate arrotondate**: scartate. Chi vuole nasconderle ha la Posizione nascosta, e i fix trovati stanno tutti vicino alla diga.
- **Il Contributore nasconde da solo la posizione di una Foto pubblicata**: scartato. La scelta si fa prima del caricamento, e un ritiro resta possibile con la mail.
- **Scelta ricordata senza chiedere di nuovo**: scartata. Chi carica spesso non sa di avere il GPS attivo.
- **Conservare il file caricato byte per byte, con il consenso**: scartata. Porterebbe su S3 numeri di serie e file pesanti, che il consenso al GPS non copre.
- **Il Backend cancella le versioni su S3**: scartato, per il permesso che richiederebbe a un processo esposto su internet.
- **Nessuna eredità del punto dall'Entità**: scartata. Mostrare il punto dell'opera è una partenza utile, e con la Posizione nascosta non rivela da dove si è scattato.

## Consequences

- Il Backend ha bisogno di credenziali S3 con scrittura sul bucket degli Originali, senza il permesso di cancellare versioni. Il bucket ha la regola che le fa scadere dopo 30 giorni.
- Serve che le mail di ritiro abbiano una risposta entro un mese: l'informativa lo dice.
- La #23 si riduce ai metadati: gli Originali, nuovi e importati, hanno già S3 con versioning.
- L'informativa privacy del forum va aggiornata per la gallery; manca già oggi con Coppermine.
- La Gallery pubblica oggi linka l'Originale su `albums/`: allo Switch il link passa alla Variante da 2048 px.
