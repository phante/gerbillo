# Ricerca della Gallery: pagina /cerca/ con griglia di Foto e FlexSearch nel browser

Deciso nel grilling del 24/10/2026, per #26. Precisa la scelta lasciata aperta da ADR 0012 («Pagefind o un indice JSON»).

La ricerca ha una pagina propria, `/cerca/?q=…`. Il campo in home, sopra la Bussola, è un modulo che porta lì e non carica nulla. La pagina scarica l'Indice di ricerca (`statico/cerca.json`, generato da `gerbillo build`), lo indicizza nel browser con FlexSearch (libreria a sé, `modelli/flexsearch.js`) e mostra:

- **Pagine**: le Entità e i Contributori il cui nome corrisponde, come schede che portano alla loro pagina.
- **Dove**: le Entità delle Foto trovate, con il numero di Foto, come pillole (le 6 più frequenti più «altre»). Una pillola restringe la griglia e finisce nell'URL (`&in=dighe/vajont`).
- **Griglia di tutte le Foto trovate**, a blocchi con «Mostra altre». Ordine per pertinenza: prima il titolo, poi Entità/Autori/Contributore, poi Tag e Categorie, poi descrizione, luogo e Datazione; a parità il pid.

I risultati si aggiornano mentre si scrive e l'URL segue la ricerca senza riempire la cronologia. La pagina della Foto non cambia: per tornare ai risultati c'è il tasto indietro. Senza JavaScript `/cerca/` rimanda a «Tutte le dighe».

Il motivo è la natura della Gallery. Sul Sito principale gli articoli sono pochi e tematici, e dieci risultati bastano. In Gallery la stessa cosa ha decine di Foto: «ponte tubo» ne trova 279 in 25 gruppi, 148 al Vajont; «coronamento» 503 in 122.

## Considered Options

- **`search.js` del sito con il suo formato di indice**: provato e scartato. Mostra 10 risultati di solo testo, utile per trovare una pagina, non per vedere le Foto. Il suo FlexSearch è chiuso nel bundle, quindi non si riusa.
- **Risultati in home al posto della Bussola**: la griglia vuole spazio, e un URL proprio si condivide sul forum.
- **Pagefind**: un secondo binario nella build (contro ADR 0012).
- **Indice precalcolato** (export di FlexSearch): si carica in 0,5 s invece di 1,5 s, ma pesa 25 MB (3,8 MB con gzip) invece di 5,4 MB (660 KB).
- **Indice a pezzi generato in Go** (un file per prefisso di parola): leggero anche da telefono, ma il confronto delle parole è da scrivere. È il ripiego se la prima ricerca dall'iPhone risulta lenta: cambia il caricamento, non la pagina.

## Consequences

- Ogni apertura di `/cerca/` scarica e indicizza tutto: circa 2 s su un computer, di più su telefono. Il ritorno da una Foto di solito è immediato grazie alla back-forward cache del browser.
- `flexsearch.js` è la 0.8 come quella del sito, ma non è una copia del sito: si aggiorna a mano.
- Rimandati: ricerca dentro la pagina della diga, pagine per Autore, Tag e Categoria.
