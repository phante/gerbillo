# Backend: binario Go, stato su file, pubblicazione con commit locale

Upload, moderazione, strumenti editoriali e generazione della Gallery pubblica (ADR 0012) stanno in un unico binario Go, statico (`CGO_ENABLED=0`). L'interfaccia del Backend è HTML generato dal server con poco JavaScript scritto a mano, incorporato con `embed`. Lo stato di lavoro (Proposte in coda, in bozza, rifiutate) è una cartella per Proposta, senza database. La Pubblicazione scrive i YAML nel Repository dei contenuti con un commit git locale (ADR 0004), poi il Backend lancia la build. L'obiettivo è un solo artefatto da distribuire e niente servizi aggiuntivi da mantenere. La storia delle modifiche viene gratis da git.

Rivisto il 15/10/2026: tolte l'interfaccia Svelte e la Pubblicazione tramite push verso un remoto.

Rivisto il 25/10/2026: l'interfaccia del Backend usa htmx e Alpine.js incorporati nel binario, al posto del poco JavaScript scritto a mano (ADR 0017).
