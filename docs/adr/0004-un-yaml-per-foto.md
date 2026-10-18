# Un file YAML per Foto come fonte di verità, in un repo git locale sul server

I metadati pubblicati di ogni Foto stanno in un file YAML proprio nel Repository dei contenuti. Il formato è disperso ma atomico: ogni modifica tocca un file solo, si versiona con git e si rivede nei diff. Non c'è un database, per ridurre le superfici dinamiche. Se un giorno servisse un file unico, si passerà a un vero database invece di un YAML monolitico.

Il Repository dei contenuti è un repo git **solo locale sul server** (rivisto il 15/10/2026). I contenuti nascono già sul server, dall'Importazione e dal Backend, quindi un giro verso un remoto e ritorno sarebbe inutile. Importazione e Backend fanno commit sul posto. Git serve da storia, diff e annullamento, non da ponte verso la build. Il backup fuori dal server è uno script esterno (#23).

## Considered Options

- **XMP incorporato nelle immagini**: scartato come fonte di verità. Le immagini restano fuori da git e i metadati non sarebbero versionabili né facili da modificare in blocco. Un convertitore YAML → XMP si farà solo se servirà davvero.
- **Repo remoto `gerbillo-contenuti` con build in Woodpecker**: scartato il 15/10/2026. Funziona per il Sito principale, dove i contenuti nascono sul Mac; qui aggiungerebbe un viaggio inutile e una dipendenza esterna.
- **SQLite**: scartato. Si perdono il diff leggibile e la storia gratuita di git.
