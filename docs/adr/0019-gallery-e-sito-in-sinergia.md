# Gallery e Sito principale lavorano in sinergia, senza vincoli

Deciso il 30/10/2026, nel grill di #62. Corregge il modello di partenza, in cui il Sito principale decideva cosa poteva esistere nella Gallery. Gli ADR che lo dicevano sono stati corretti o rimossi: la versione vecchia è nella storia git.

- **La Gallery decide cosa ha una pagina nella Gallery.** Nessun soggetto aspetta il sito, e nessuna regola della Gallery si giustifica con «il sito non ce l'ha».
- **Cos'è un'Entità.** Un soggetto con un nome proprio: un'opera, un evento, un reportage. Il test è «si può dire *la X*?»: «la Fiera Primiero 2008» sì, «le turbine Pelton» no. L'Entità dice chi è il soggetto, la Categoria di che genere è, e una Foto può avere entrambe.
- **Due fonti, unite al build.** I file `entita/<sezione>/<nome>.yaml` del Repository dei contenuti per ciò che ha solo la Gallery, e l'Indice delle Entità per ciò che ha una pagina sul sito. Se uno Slug è in entrambi, i dati del sito vincono. Se il sito toglie una pagina, il build lo segnala e un Moderatore crea il file. Ogni Entità nella Gallery ha il proprio Slug `sezione/nome`, con o senza pagina sul sito.
- **Sinergia.** Se il sito ha lo stesso Slug, la pagina della Gallery prende link e scheda dati. Il formato comune di Slug e chiavi (ADR 0013) evita traduzioni, e i due indici restano l'unico canale (ADR 0002).
- **Sezioni.** La Gallery crea Entità in `dighe/`, `centrali/`, `opere/` (opere idrauliche: la Categoria distingue presa, canale, condotta) ed `eventi/`. `reportage/`, `tecnica/` e ogni altra sezione arrivano dal sito e sono accettate senza toccare il codice.
- **Informazioni interne.** Ciò che serve solo a Moderatori e amministratori resta nel Backend, e il build pubblico non lo scrive mai in `dist/`. Un'informazione nuova nasce interna, salvo decisione contraria. Non avere una pagina sul sito è un'informazione interna: nella Gallery pubblica la pagina si riconosce solo perché manca il link al sito.
- **Foto senza Entità.** Una Foto con una Categoria è classificata, e si trova nella Vista per Categoria. Una senza Entità e senza Categoria è «Da classificare» solo nel Backend: la Gallery pubblica la mostra in «Altre foto», senza raccontare il lavoro da fare. Oggi la pagina divide per categoria di Coppermine; allo Switch diventa una sola (#40).

## Considered Options

- **Il sito come riferimento, con la Gallery che aggiunge solo ciò che il sito avrà**: scartato. È il modello che ha prodotto un secondo concetto di Entità, con URL e campi propri, per ciò che poteva essere una pagina come le altre.
- **Un file per ogni Entità nella Gallery, anche quelle del sito**: scartato. Indipendenza totale, ma centinaia di file doppi da tenere allineati, per un caso raro: una pagina tolta dal sito.
- **Qualunque raccolta di Foto come Entità, anche le turbine**: scartato. La Categoria esprime già il genere, e il confine fra le due sparirebbe.
- **Solo opere con un luogo come Entità**: scartato. Un evento ha un nome proprio e merita la sua pagina.

## Consequences

- Il codice e le issue si allineano in passi successivi (ADR 0018, #62): finché non è fatto, il codice ha ancora `da_censire` e `/da-censire/`.
- La Vista per Categoria non esiste ancora: fino ad allora quelle Foto si trovano con la ricerca.
- L'unica dipendenza che resta è la pagina tolta dal sito, che il build segnala.
