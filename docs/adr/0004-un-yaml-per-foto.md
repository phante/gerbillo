# Un file YAML per Foto come fonte di verità, nessun database

I metadati pubblicati di ogni Foto stanno in un file YAML proprio nel Repository dei contenuti. Il formato è disperso ma atomico: ogni modifica tocca un file solo, si versiona con git e si rivede nei diff. Non c'è un database, per ridurre le superfici dinamiche. Se un giorno servisse un file unico, si passerà a un vero database invece di un YAML monolitico.

## Considered Options

- **XMP incorporato nelle immagini**: scartato come fonte di verità. Le immagini restano fuori da git e i metadati non sarebbero versionabili né facili da modificare in blocco. Un convertitore YAML → XMP si farà solo se servirà davvero.
