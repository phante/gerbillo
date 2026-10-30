#!/usr/bin/env python3
"""Prova di Nominatim per le aree amministrative delle Entità senza pagina (ADR 0018).

Per un'Entità del sito per ogni provincia (e per ogni stato estero) chiede a Nominatim il
reverse geocoding del Punto geografico e confronta stato, regione e provincia con i valori
del sito. Rispetta la policy d'uso: una richiesta ogni 1,1 s, User-Agent con il contatto,
nessuna ripetizione. Circa 100 richieste, meno di due minuti.

Uso, dalla root del repo (con l'Indice delle Entità di testdata/sito):
  python3 scripts/prova-nominatim.py > nominatim.tsv
Stampa una riga per Entità: valori del sito, valori di Nominatim (county, state, country),
codici ISO 3166-2 e se coincidono. Il riepilogo va su stderr.
"""
import json, subprocess, sys, time, urllib.parse, urllib.request

UA = "Gerbillo/0.1 (gallery di progettodighe.it; info@progettodighe.it)"

def indice():
    out = subprocess.run(["go", "run", ".", "indice-entita", "-sito", "../testdata/sito"],
                         cwd="backend", capture_output=True, text=True, check=True).stdout
    return json.loads(out)

def campione(idx):
    visti, scelte = set(), []
    for e in idx:
        g = e.get("geolocation", {})
        c = g.get("coords")
        if not c:
            continue
        chiave = (g.get("stato", ""), g.get("provincia", "") if g.get("stato") == "Italia" else "")
        if chiave not in visti:
            visti.add(chiave)
            scelte.append(e)
    return scelte

def reverse(lat, lon):
    q = urllib.parse.urlencode({"format": "jsonv2", "lat": lat, "lon": lon, "zoom": 10,
                                "addressdetails": 1, "accept-language": "it"})
    req = urllib.request.Request("https://nominatim.openstreetmap.org/reverse?" + q, headers={"User-Agent": UA})
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.load(r).get("address", {})

def main():
    righe = campione(indice())
    print("\t".join(["slug", "sito stato", "sito regione", "sito provincia", "county", "state", "country",
                     "ISO lvl4", "ISO lvl6", "regione uguale", "provincia uguale", "address"]))
    uguali = [0, 0]
    for i, e in enumerate(righe):
        g, c = e["geolocation"], e["geolocation"]["coords"]
        if i:
            time.sleep(1.1)
        try:
            a = reverse(c["latitudine"], c["longitudine"])
        except Exception as err:
            print(e["slug"], "errore", err, sep="\t")
            continue
        reg, prov = a.get("state", ""), a.get("county", "")
        r_ok, p_ok = reg == g.get("regione", ""), prov == g.get("provincia", "")
        uguali[0] += r_ok
        uguali[1] += p_ok
        print("\t".join([e["slug"], g.get("stato", ""), g.get("regione", ""), g.get("provincia", ""), prov, reg,
                         a.get("country", ""), a.get("ISO3166-2-lvl4", ""), a.get("ISO3166-2-lvl6", ""),
                         str(r_ok), str(p_ok), json.dumps(a, ensure_ascii=False)]), flush=True)
    print(f"{len(righe)} Entità: regione uguale {uguali[0]}, provincia uguale {uguali[1]}", file=sys.stderr)

if __name__ == "__main__":
    main()
