# Backend: binario Go con interfaccia Svelte incorporata, stato su file, pubblicazione via git

Upload, moderazione e strumenti editoriali stanno in un unico binario Go. L'interfaccia Svelte è compilata e incorporata con `embed` e condivide i componenti con la Gallery pubblica. Lo stato di lavoro (Proposte in coda, in bozza, rifiutate) è una cartella per Proposta, senza database. La Pubblicazione scrive i YAML nel Repository dei contenuti con un commit git. L'obiettivo è un solo artefatto da distribuire e niente servizi aggiuntivi da mantenere. La storia delle modifiche viene gratis da git.
