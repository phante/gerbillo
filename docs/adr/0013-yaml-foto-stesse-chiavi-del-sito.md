# Il YAML della Foto usa le stesse chiavi del Sito principale

Deciso il 23/10/2026, prima di G2 (#10); corretto il 30/10/2026 (ADR 0019).

Dove un concetto esiste anche nel front matter del Sito principale, la Foto usa la stessa chiave e lo stesso significato. È una scelta di formato, non di dipendenza: la Gallery non chiede al sito cosa può esistere. Le chiavi proprie della gallery restano in italiano, come fa il sito con `corsoacqua` o `revisione`. Un solo modello mentale per un solo manutentore, e l'integrazione con il sito (Indice delle Foto, Foto usate nelle pagine) non richiede traduzioni.

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

- **`slug`**: le Entità che la Foto ritrae, come percorso `sezione/nome` (`dighe/agaro`, `centrali/goglio-baceno-vb`). Si valida contro le Entità della Gallery: i file di `entita/` più l'Indice delle Entità (ADR 0019). I campi `dighe:` e `centrali:` del sito restano suoi: sono gli archi del grafo da cui nasceranno gli Impianti.
- **`tags`**: come nel sito, per i collegamenti diretti con le sue pagine tag. La gallery può aggiungerne di propri. Vocabolario controllato nel Livello di arricchimento.
- **`categories`**: tipo o sottotipo del soggetto, con valori a percorso (`tecnica/turbine`). Nel sito `categories` è in dismissione, quindi non c'è conflitto.
- **`date`**: la Data di caricamento, come la `date` delle pagine e delle immagini del sito.
- **`authors`**: l'Autore, una lista di persone o enti.
- **`datazione`**: quando è stata scattata, in EDTF (ISO 8601-2): `2009-06-03`, `2009-06`, `2009`, `195X`, `1938/1940`, `1950~`. `date` non può esprimere la precisione.
- **`geolocation`**: stesse chiavi del sito. `coords` è il punto geografico centrale associato alla Foto, come nel sito lo è per l'articolo. I campi mancanti si ereditano al build dalla prima Entità in `slug`; `map_ids` si eredita sempre e non si scrive mai nella Foto. `zoom` non serve.
- **`credits`** non si salva: si genera da `authors`, `datazione` e Fonte quando una Foto finisce nel sito.

## Considered Options

- **Chiavi proprie della gallery, con una traduzione verso il sito**: scartato. Due modelli mentali per lo stesso manutentore.
- **Slug senza sezione**: scartato. Il percorso porta con sé il tipo e non dipende dall'unicità dei nomi fra sezioni.
- **`dighe:` e `centrali:` come nel sito**: scartato. Servono al grafo degli Impianti, e una sezione nuova richiederebbe un campo nuovo.

## Consequences

- I YAML di G1 rinominano `titolo`, `descrizione`, `caricata` e `autore` in `title`, `description`, `date` e `authors`.
- I valori dei tag del sito oggi sono pochi e disordinati: la gallery li eredita dove esistono e aiuta a sistemarli.
- Lo spostamento di una pagina nel sito cambia uno Slug: il build lo rileva, e si risolve con un alias (ADR 0018).
