# Modello entità + attributi, l'albero è solo una Vista

**Rivisto dall'ADR 0013** (23/10/2026): il collegamento è il campo `slug` con percorsi `sezione/nome`, il Tema diventa `tags` e `categories` come nel Sito principale. Resta valido il principio: l'albero è solo una Vista.

Coppermine impone un albero rigido di Categorie e Album. Gerbillo collega invece ogni Foto a una o più Entità e la descrive con Attributi (Tema, Datazione, Autore, Fonte). L'albero regione → diga → tema è una Vista generata, alla pari di quelle per periodo, per mappa o per Tema. Così una foto che ritrae due dighe non va duplicata o forzata in un solo album. Resta comunque l'impostazione comunitaria e strutturata, che va preservata.
