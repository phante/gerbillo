# Correzioni degli utenti con coda di moderazione

Deciso il 24/10/2026, nel grilling di #5. Arriva con lo Switch: prima non ci sono Contributori, e le Foto le corregge solo un Moderatore.

Chiunque abbia una sessione del forum può proporre una Correzione a una Foto pubblicata, dal link "Segnala una correzione" della pagina della Foto. Il link apre nel Backend un modulo con il pid già compilato. La Correzione è strutturata: valori nuovi per i campi, con una nota facoltativa. Il Moderatore la approva o la rifiuta con un clic, senza riscrivere a mano ciò che l'utente ha già detto.

- **Campi.** Tutti quelli editoriali: `title`, `description`, `alt`, `slug`, `categories`, `tags`, `datazione`, `authors`, Fonte, `geolocation.coords`. Restano fuori `pid`, `date`, `contributore`, `formato`, le dimensioni e la Licenza (ADR 0010). Sostituire il file è un caricamento nuovo.
- **Chi salta la coda.** I Moderatori, su tutte le Foto. Il gruppo phpBB *Gallery unlimited*, solo sulle proprie. Tutti gli altri passano dalla coda, anche il Contributore sulle sue Foto: un nuovo arrivato potrebbe caricare una Foto lecita e poi infilarci un link di spam. Il gruppo segue la stessa logica del forum, dove i primi 5 messaggi dei nuovi arrivati si approvano uno per uno.
- **Eccezioni.** Finché la sua Proposta non è pubblicata, chi carica la corregge da solo. Nascondere la posizione di una propria Foto resta immediato (ADR 0015).
- **Coda.** Il Moderatore decide campo per campo e può ritoccare un valore prima di approvarlo. Per ogni campo toccato la Correzione salva il valore di partenza. Se il campo è cambiato nel frattempo, la coda lo segnala e mostra le due versioni. Le Correzioni sulla stessa Foto compaiono raggruppate. Nessun blocco: decide il Moderatore.
- **Rifiuto.** Il motivo si sceglie fra quelli già pronti ("non verificabile", "spam", "doppione") più un testo libero, ed è facoltativo. Le Correzioni rifiutate si tengono.
- **Antispam.** Al massimo circa 20 Correzioni aperte per utente. Un'etichetta in coda sulle Correzioni che aggiungono un URL. Per il Moderatore, l'azione "rifiuta tutte le aperte di questo utente". Nessun filtro blocca l'invio.
- **Riscontro.** Una pagina "Le mie Correzioni" nel Backend, con stato e motivo del rifiuto. Niente messaggi privati sul forum.
- **Storia.** L'approvazione è un commit nel Repository dei contenuti: autore chi ha proposto, committer il Moderatore, e l'id della Correzione in un trailer.
- **Avvisi.** Per i Moderatori un contatore nel Backend. Per l'admin un riepilogo Telegram una volta al giorno, solo se la coda non è vuota.

## Considered Options

- **Segnalazione in testo libero**: scartata come forma principale. Lascia al Moderatore tutto il lavoro di trascrivere la correzione. Resta possibile come Correzione con la sola nota.
- **Thread del forum invece di un modulo**: scartato. Un thread non ha stato, e l'admin dovrebbe ricordare cosa ha già sistemato.
- **Approvazione solo in blocco**: scartata. Spesso una parte della Correzione è giusta e un'altra no.
- **Il Contributore modifica subito le sue Foto**: scartato per tutti tranne *Gallery unlimited*, per il rischio spam.
- **Messaggio privato sul forum**: scartato. Il bridge (ADR 0008) dovrebbe scrivere su phpBB.
- **Avviso via mail**: rimandato. Servirebbe un SMTP nel Backend; per ora basta Telegram.

## Consequences

- Ogni Foto ha come Contributore un account del forum. Anche il bot Telegram carica a nome di un utente collegato (issue a parte).
- Gli account si cancellano a mano dalla console del forum, quindi può sparire anche uno con Foto o Correzioni. Gerbillo lo tollera: la Foto tiene `contributore.nome` e resta pubblicata, e le Correzioni ancora aperte si chiudono come rifiutate. Alcune Foto di Coppermine hanno già perso il Contributore: l'Importazione (#9) recupera il nome se Coppermine lo conserva, altrimenti il Contributore è sconosciuto.
- Il Backend deve sapere se l'utente è in *Gallery unlimited*: lo dice già il campo `groups` del bridge.
