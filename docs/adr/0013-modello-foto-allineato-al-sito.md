# Il YAML della Foto usa il modello dati del Sito principale

Rivede l'ADR 0003. Deciso il 23/10/2026, prima di G2 (#10).

Dove un concetto esiste già nel front matter del Sito principale, la Foto usa la stessa chiave e lo stesso significato. Le chiavi proprie della gallery restano in italiano, come fa il sito con `corsoacqua` o `revisione`. Un solo modello mentale per un solo manutentore, e l'integrazione con il sito (Indice delle Foto, Foto usate nelle pagine) non richiede traduzioni.

```yaml
pid: 18
title: Il lago a monte della diga
description: Il bacino di Agaro visto dal coronamento…
alt: Specchio d'acqua turchese…
date: 2006-12-27T12:23:55+01:00
authors: [bennygreco]
datazione: 2009-06
slug: [dighe/agaro]
tags: [toce]
categories: [paesaggio/lago]
geolocation:
  stato: Italia
  regione: Piemonte
  provincia: Verbano-Cusio-Ossola
  coords: {latitudine: 46.2905, longitudine: 8.2987}
contributore: {id: 25, nome: …}
formato: immagine
larghezza: 2048
altezza: 1153
```

- **`slug`**: le pagine del sito che la Foto ritrae, come percorso `sezione/nome` (`dighe/agaro`, `centrali/goglio-baceno-vb`). Vale per qualunque sezione, anche reportage e tecnica, e si valida contro l'Indice delle Entità. I campi `dighe:` e `centrali:` del sito restano suoi: sono gli archi del grafo da cui nasceranno gli Impianti.
- **`tags`**: come nel sito, per i collegamenti diretti con le sue pagine tag. La gallery può aggiungerne di propri. Vocabolario controllato nel Livello di arricchimento.
- **`categories`**: tipo o sottotipo del soggetto, con valori a percorso (`tecnica/turbine`). Nel sito `categories` è in dismissione, quindi non c'è conflitto. Sostituisce il Tema.
- **`date`**: la Data di caricamento, come la `date` delle pagine e delle immagini del sito.
- **`authors`**: l'Autore, una lista di persone o enti.
- **`datazione`**: quando è stata scattata, in EDTF (ISO 8601-2): `2009-06-03`, `2009-06`, `2009`, `195X`, `1938/1940`, `1950~`. `date` non può esprimere la precisione.
- **`geolocation`**: stesse chiavi del sito. `coords` è il punto geografico centrale associato alla Foto, come nel sito lo è per l'articolo. I campi mancanti si ereditano al build dalla prima pagina in `slug`; `map_ids` si eredita sempre e non si scrive mai nella Foto. `zoom` non serve.
- **`credits`** non si salva: si genera da `authors`, `datazione` e Fonte quando una Foto finisce nel sito.

## Considered Options

- **Entità e Temi propri della gallery (ADR 0003)**: scartato. Richiedeva un tipo di Entità e una traduzione verso il sito, e non copriva reportage e tecnica.
- **Slug senza sezione**: scartato. Il percorso porta con sé il tipo e non dipende dall'unicità dei nomi fra sezioni.
- **`dighe:` e `centrali:` come nel sito**: scartato. Servono al grafo degli Impianti, e una sezione nuova richiederebbe un campo nuovo.

## Consequences

- I YAML di G1 rinominano `titolo`, `descrizione`, `caricata` e `autore` in `title`, `description`, `date` e `authors`.
- I valori dei tag del sito oggi sono pochi e disordinati: la gallery li eredita dove esistono e aiuta a sistemarli.
- Lo spostamento di una pagina nel sito rompe i riferimenti: il build lo rileva contro l'Indice delle Entità.
- Sinergia con il sito, ma la gallery non si blocca ad aspettarlo: dove il sito non ha una pagina o un valore, la Foto resta senza `slug` o usa un valore proprio di `categories`, e si collega quando il sito lo avrà (#1).

Integrato il 25/10/2026: un'opera che non ha ancora una pagina nel sito è un'Entità da censire, nel campo `da_censire`, accanto a `slug` (ADR 0018). `slug` resta validato contro l'Indice delle Entità.
