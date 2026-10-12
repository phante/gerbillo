# Durante la convivenza Coppermine resta la fonte primaria

Fino allo Switch gli utenti continuano a usare Coppermine. L'Importazione è idempotente e incrementale e si può rieseguire in qualsiasi momento. Il lavoro editoriale fatto in Gerbillo (Mappature, correzioni dei caratteri corrotti, arricchimenti) vive in un Livello di arricchimento separato, applicato sopra ogni Importazione, così non viene mai sovrascritto. Così si evita un congelamento dell'archivio lungo quanto la migrazione e si può fermare il progetto in ogni momento senza perdite.
