# Autenticazione tramite estensione bridge di phpBB

Il Backend non ha utenti propri: riusa la sessione del forum. Un'estensione phpBB (`ext/…`) espone un controller raggiungibile solo da `127.0.0.1`. Il controller risponde con JSON versionato: `{v, authenticated, user_id, username, groups}`. Il Backend lo chiama quando non ha una sessione propria valida, poi crea una propria sessione di breve durata. I Moderatori sono definiti dai gruppi phpBB. Il Backend dipende da una propria interfaccia "utente corrente", non da phpBB direttamente.

## Considered Options

- **Script PHP sciolto**: scartato, perché non sopravvive agli upgrade del forum. Un'estensione sì.
- **`auth_request` di nginx**: valutato e scartato a favore della chiamata diretta dal Backend.

## Consequences

- Il percorso dei cookie di phpBB deve essere `/`.
- Il bridge non crea sessioni nuove: se la sessione non esiste risponde "ospite". `session_begin()` in una sottorichiesta con "ricordami" creerebbe sessioni a ripetizione, perché i `Set-Cookie` vanno persi.
- `app.php` di phpBB chiama comunque `session_begin()` prima di ogni controller. Perciò il Backend non inoltra il cookie `_k`, e il controller legge la sessione del cookie `_sid` direttamente dalla tabella: la chiamata viene da `127.0.0.1`, e `session_begin()` non riconoscerebbe l'IP del browser. Ogni chiamata lascia una sessione da ospite su `127.0.0.1`, che phpBB pulisce da solo (#66).
- La sessione propria del Backend sta in memoria, ha come chiave il `_sid` del forum e dura 15 minuti. È il ritardo massimo con cui il Backend vede un cambio di gruppo o un ban.
