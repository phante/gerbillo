# Autenticazione tramite estensione bridge di phpBB

Il Backend non ha utenti propri: riusa la sessione del forum. Un'estensione phpBB (`ext/…`) espone un controller raggiungibile solo da `127.0.0.1`. Il controller risponde con JSON versionato: `{v, authenticated, user_id, username, groups}`. Il Backend lo chiama quando non ha una sessione propria valida, poi crea una propria sessione di breve durata. I Moderatori sono definiti dai gruppi phpBB. Il Backend dipende da una propria interfaccia "utente corrente", non da phpBB direttamente.

## Considered Options

- **Script PHP sciolto**: scartato, perché non sopravvive agli upgrade del forum. Un'estensione sì.
- **`auth_request` di nginx**: valutato e scartato a favore della chiamata diretta dal Backend.

## Consequences

- Il percorso dei cookie di phpBB deve essere `/`.
- Il bridge non crea sessioni nuove: se la sessione non esiste risponde "ospite". `session_begin()` in una sottorichiesta con "ricordami" creerebbe sessioni a ripetizione, perché i `Set-Cookie` vanno persi.
