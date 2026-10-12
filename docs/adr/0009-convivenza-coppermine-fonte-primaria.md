# Durante la convivenza Coppermine resta la fonte primaria

Fino allo Switch gli utenti continuano a usare Coppermine. L'Importazione è idempotente: ogni esecuzione rigenera da zero i dati importati e si può rieseguire in qualsiasi momento. Il lavoro editoriale fatto in Gerbillo (Mappature, correzioni, arricchimenti) vive in un Livello di arricchimento separato, applicato sopra ogni Importazione, così non viene mai sovrascritto. Così si evita un congelamento dell'archivio lungo quanto la migrazione e si può fermare il progetto in ogni momento senza perdite.

I caratteri corrotti non sono lavoro editoriale: li corregge l'Importazione con regole automatiche. I campi sono salvati in UTF-8 dentro colonne latin1, tranne i caricamenti recenti, che sono in cp1252. Le entità HTML sono salvate escapate. Il Livello di arricchimento resta per le correzioni di contenuto.
