# Gallery pubblica statica in Astro + Svelte, separata dal Sito principale

**Sostituito dall'ADR 0012** (15/10/2026): la Gallery pubblica la genera il binario Go. Resta valida la separazione dal Sito principale.

La Gallery pubblica è un sito statico generato con Astro e isole Svelte, separato dal Sito principale in Hugo. Lo scopo è non avere un backend sempre acceso per servire le pagine. Il manutentore è uno solo e ha poche ore a settimana, e le intrusioni passate erano entrate da Coppermine. Astro non è una scelta ovvia, perché il manutentore conosce Hugo e non Astro. È stato preferito per i componenti Svelte condivisi con il Backend e per la ricerca lato client. La sperimentazione è accettata.

## Consequences

Il gate G0 del pilota verifica la scelta: build di circa 25.000 YAML sintetici in meno di 5 minuti circa, con memoria compatibile con il server Lightsail. **Se G0 fallisce, si torna a Hugo.**

**Esito G0 (ottobre 2026): superato.** Una build unica di 25.000 Foto sintetiche dura meno di 30 secondi, ma la sua memoria cresce con il numero di Foto (circa 340 MB fissi più 20 MB ogni 1000 Foto). Per questo la build si fa a blocchi: un indice leggero di tutte le Foto, poi più esecuzioni di Astro su gruppi di Foto. Così la memoria resta sotto i 400 MB anche con 100.000 Foto, al prezzo di una build più lenta. Misure e prototipo nel branch `prototype/g0-astro-build` (issue #8).
