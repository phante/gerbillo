# Gallery pubblica generata dal binario Go, senza Astro

Deciso nel grilling del 15/10/2026, scritto il 18/10/2026. L'ADR 0001 su Astro è stato rimosso: resta nella storia git.

La Gallery pubblica resta un sito statico servito da nginx, ma la genera lo stesso binario del Backend: `gerbillo build` scrive `dist/` con `html/template`. Il binario ha tre sottocomandi, `importa`, `build` e `serve`. Il JavaScript è scritto a mano e ridotto al minimo: il visualizzatore, "Copia per il forum" e una mappa Leaflet senza framework. La ricerca è in ADR 0014.

Il motivo è il flusso. I metadati nascono sul server (ADR 0004) e la build la lancia il Backend dopo ogni Pubblicazione. Con Astro il Backend dovrebbe avviare un container Node, perché Node non va installato sull'host. Per farlo gli servirebbe `docker.sock`, cioè di fatto root sull'host, dato proprio al processo esposto su internet. Con Go la build è una funzione dello stesso binario: un solo linguaggio, niente Node, niente build a blocchi.

## Considered Options

- **Astro + Svelte**: scartato per i motivi sopra. Il gate G0 (#8) l'aveva superato, ma servivano la build a blocchi per la memoria e un container Node.
- **Hugo**: è noto, ma servirebbero il binario `hugo` sull'host o di nuovo Docker, due linguaggi e una conversione dei dati.

## Consequences

- Le pagine pubbliche non dipendono dal processo del Backend: se il Backend cade, `dist/` resta servito.
- Niente componenti condivisi tra Backend e Gallery pubblica, e l'interfaccia è meno ricca.
- La build a blocchi (#14) non serve più. Resta da misurare il tempo di `gerbillo build` su tutto l'archivio.
