# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Due pubblici guidano le decisioni di design:

- **Visitatori della Gallery pubblica.** Appassionati della community di ProgettoDighe (utenti del forum che cercano e condividono foto di opere note) e pubblico generico che arriva dalla ricerca web o dal Sito principale su un'opera precisa. Consultano: cercano un'Entità, sfogliano le sue Foto, ne copiano una per il forum.
- **Moderatori.** Amministratori e moderatori del forum che usano il Backend come strumento di redazione: approvano Proposte e Correzioni, classificano, collegano Entità e fanno la sistemazione massiva dell'archivio prima dello Switch da Coppermine. Lavoro lungo e ripetitivo, spesso dal telefono.

I Contributori (utenti del forum che caricano Foto) sono un pubblico secondario: il caricamento deve funzionare bene, ma non guida le scelte.

## Product Purpose

Gerbillo è la gallery fotografica della community di ProgettoDighe e sostituisce Coppermine. Le Foto documentano dighe, centrali, altre opere idrauliche ed eventi. Successo: l'archivio storico migra senza perdere pid e URL, i visitatori trovano le Foto di un'opera, e i Moderatori riescono a sistemare l'archivio con uno strumento comodo abbastanza da essere usato davvero.

## Positioning

Un archivio fotografico di opere idrauliche italiane tenuto da una community, organizzato per Entità con nome proprio («la diga di X») e collegato al Sito principale tramite lo stesso Slug. Non è una gallery social: niente commenti né voti, la discussione vive nel forum (ADR 0006).

## Operating Context

- Gallery pubblica: sito statico generato dal binario Go (`gerbillo build`, `html/template`), servito da nginx, JavaScript minimo scritto a mano (visualizzatore, Copia per il forum, mappa Leaflet, ricerca FlexSearch).
- Backend: HTML dal server Go con htmx e Alpine.js, librerie incorporate nel binario, niente build JS, niente CDN. Pensato prima per il telefono. Non deve funzionare senza JavaScript.
- Autenticazione tramite la sessione del forum phpBB (ADR 0008).
- Vocabolario di dominio in `CONTEXT.md` (Foto, Entità, Variante, Proposta, Correzione, Moderatore…): l'interfaccia usa questi termini.
- Lingua dell'interfaccia: italiano.

## Capabilities and Constraints

- Vincoli tecnici e decisioni negli ADR in `docs/adr/`.
- Informazioni interne (tag interni, «Da classificare», Entità senza pagina sul sito) non compaiono mai nella Gallery pubblica.
- Nessun commento né voto.
- Test end-to-end Playwright in `e2e/`, a larghezza telefono.

## Brand Commitments

- Nome pubblico: Gallery ProgettoDighe. Logo `backend/modelli/sito/PDLogo.svg`.
- Oggi la Gallery pubblica riprende l'aspetto del Sito principale (Tachyons, `main.css`, header e footer di progettodighe.it, `backend/modelli/sito/`). È il riferimento attuale, ma rivedibile: un'evoluzione visiva della Gallery è possibile.
- Il Backend usa colori e font della Gallery pubblica ma un layout da strumento: non deve sembrare una sezione del sito (ADR 0017).

## Evidence on Hand

- Archivio reale importato da Coppermine (Foto, Album, circa 670 commenti storici da rivedere).
- Dati di prova pubblici in `testdata/`.
- Nessuna testimonianza, statistica d'uso o metrica pubblicata: non inventarle.

## Product Principles

1. Le Foto e le opere vengono prima dell'interfaccia: la Gallery pubblica è consultazione, non vetrina.
2. Il Backend è uno strumento di lavoro: veloce, denso, comodo dal telefono, altrimenti non si usa.
3. Stabilità: pid e URL non cambiano mai.
4. Ciò che è interno resta interno; nel dubbio un'informazione nasce interna.
5. La discussione appartiene al forum; la Gallery facilita il passaggio (Copia per il forum).
