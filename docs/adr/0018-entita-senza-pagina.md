# Entità senza pagina, con Slug generato e alias

Deciso il 25/10/2026 e riscritto il 30/10/2026, nel grill di #62 (ADR 0019): come cataloga una Foto chi non conosce il sito, e cosa succede se l'opera non ha una pagina sul sito.

Molte opere non hanno una pagina sul sito: l'Importazione ha 293 album di Coppermine non collegati, con 7431 Foto, spesso intitolati a opere. Sono Entità come le altre.

- **Entità senza pagina.** Un'Entità che la Gallery ha e il sito no. Sta in `entita/<sezione>/<nome>.yaml`, con `title`, `geolocation` e `aliases`. Le Foto la indicano con `slug`, come tutte. Non esiste un campo a parte, né un URL a parte: `/dighe/verzasca/` è una pagina come le altre, senza il link al sito. Il filtro «Senza pagina» sta solo nel Backend (ADR 0019).
- **«Cosa ritrae?».** Chi carica sceglie l'Entità per nome, tipo e comune, o tra le più vicine al GPS (#29). Se non c'è, ne crea una con nome, tipo e punto, dopo che il modulo ha proposto quelle simili per nome e distanza. Il sistema genera lo Slug, che chi carica non vede mai: il tipo dà la sezione, il nome dà il resto. Se lo Slug è occupato prende un numero, `dighe/verzasca-2`.
- **Chi crea.** Segue la Foto (ADR 0016): Moderatori e *Gallery unlimited* subito, gli altri con la Proposta, e l'Entità diventa pubblica all'approvazione.
- **Il sito pubblica lo stesso Slug.** Non serve nessuna azione: i dati del sito vincono, la pagina prende il link al sito ed esce da «Senza pagina». Il file in `entita/` resta innocuo, e il build lo segnala perché un Moderatore lo tolga.
- **Il sito pubblica uno Slug diverso.** Quando nell'Indice delle Entità c'è una pagina con nome simile entro 5 km (senza punto, solo se il nome è certo), il Backend propone «Collega»: tutte le Foto prendono lo Slug nuovo con un commit solo, e quello vecchio diventa un alias.
- **Alias.** Come in Hugo: `aliases` nell'Entità elenca gli URL vecchi, e il build scrive per ciascuno una pagina di rimando (meta refresh e `canonical`, perché una pagina statica non può rispondere 301). Ogni alias punta direttamente all'URL attuale, senza catene. La scheda del Backend li mostra. Se un alias coincide con una pagina vera, vince la pagina e il build lo segnala. Vale la stessa regola per collegamento, rinomina e unione di doppioni.
- **Doppioni.** I Moderatori rinominano e uniscono con un commit solo; l'Entità tolta resta come alias di quella tenuta.
- **Dove si scrive.** Il Backend scrive sempre in `entita/<sezione>/<nome>.yaml`, e Annulla (#59) copre quella cartella. Nell'anteprima i file seme si spostano lì all'avvio.
- **Album non collegati.** «Crea dall'album», uno per uno o in blocco, deciso da un Moderatore: molti titoli non sono nomi propri («Varie», gli album personali). Sezione proposta dalla categoria di Coppermine (Meeting → `eventi/`), aree dal percorso quando si riconoscono, tutto correggibile nell'anteprima prima di confermare.
- **Aree amministrative.** Stato, regione e provincia li propone Nominatim (OpenStreetMap) dal Punto geografico, quando si crea l'Entità o se ne sposta il punto: dal server, mai dal browser né dal build, con una coda di al massimo una richiesta al secondo e uno User-Agent con il contatto, come chiede la policy d'uso. Nominatim aiuta a compilare ma non è la fonte: il valore a mano vince, e spostando il punto i valori nuovi si propongono senza sovrascrivere. Se Nominatim non risponde, i campi restano vuoti e c'è «Ricalcola dal punto»: il caricamento non dipende da un servizio esterno. I nomi si allineano a quelli dell'Indice delle Entità, senza «Provincia di» o «Città metropolitana di»: si prende il nome di Nominatim in italiano (`county`, `state`, `country`) e si tolgono i prefissi solo se compaiono. La prova di #60 (88 Entità, una per provincia) li ha dati già puliti, con 85 regioni e 81 province uguali al sito; le differenze sono errori o dati vecchi del sito, e province che Nominatim non ha (Aosta, Massa-Carrara).
- **Attribuzione.** Una riga per OpenStreetMap e Nominatim nel footer della Gallery pubblica e accanto ai campi nel Backend. La pagina Licenze del Sito principale si aggiorna a parte (#64).
- **Categorie per chi carica.** Facoltative, da etichette leggibili: ogni voce del vocabolario diventa `{valore, etichetta}` in `dati/vocabolario.yaml`.

## Considered Options

- **Entità da censire, con id numerico, campo `da_censire` e URL `/da-censire/<id>/`**: la prima versione di questo ADR, scartata il 30/10/2026. Separava dalle altre Entità un soggetto che è uguale a loro, ed esponeva al pubblico un'informazione interna.
- **Nome libero sulla singola Foto**: scartato. I Moderatori dovrebbero raggruppare a mano lo stesso nome scritto in modi diversi, e il secondo utente non troverebbe l'opera già segnalata.
- **Foto in coda finché l'opera non ha una pagina sul sito**: scartata. La coda serve a controllare i contenuti, non la classificazione, e la pagina può non arrivare mai.
- **`da-censire.json` pubblico per il sito**: scartato. L'elenco delle Entità senza pagina è un'informazione interna, e sta nel Backend.
- **Suffisso con la provincia per gli omonimi** (`dighe/verzasca-so`): scartato. La provincia può mancare o cambiare; il numero funziona sempre.
- **Aree amministrative dall'Entità del sito più vicina**: scartata. Vicino non vuol dire nella stessa area: un crinale fa confine, e molte centrali sono in pianura, lontane dalle dighe.
- **Nomi dai codici ISO 3166-2** (`ISO3166-2-lvl6` con una tabella codice → nome): scartati. Il codice di provincia manca per Aosta e per l'estero, quindi serve comunque il nome; la tabella non aggiunge nulla.
- **Confini amministrativi nel binario**: scartati. Pesano qualche MB e vanno aggiornati a ogni riforma delle province.
