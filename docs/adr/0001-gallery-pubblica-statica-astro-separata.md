# Gallery pubblica statica in Astro + Svelte, separata dal Sito principale

La Gallery pubblica è un sito statico generato con Astro e isole Svelte, separato dal Sito principale in Hugo. Lo scopo è non avere un backend sempre acceso per servire le pagine. Il manutentore è uno solo e ha poche ore a settimana, e le intrusioni passate erano entrate da Coppermine. Astro non è una scelta ovvia, perché il manutentore conosce Hugo e non Astro. È stato preferito per i componenti Svelte condivisi con il Backend e per la ricerca lato client. La sperimentazione è accettata.

## Consequences

Il gate G0 del pilota verifica la scelta: build di circa 25.000 YAML sintetici in meno di 5 minuti circa, con memoria compatibile con il server Lightsail. **Se G0 fallisce, si torna a Hugo.**
