#!/usr/bin/env python3
"""Anonimizza l'estrazione di Coppermine (grezzo/coppermine.tsv.gz) in testdata/coppermine.sql.gz.

- username e Autore (user1): ogni parola diventa una parola finta con HMAC e una chiave
  casuale che non viene mai salvata. Stessa parola, stessa parola finta, in entrambi i
  campi. Iniziale, maiuscole, punteggiatura, numeri ed entità HTML restano.
- exifData: resta solo la data di scatto (Coppermine non salva il GPS).
- album riservati (visibility diversa da 0): spariscono con le loro Foto ed EXIF. Il dump è
  pubblico, e filepath e filename porterebbero ai file privati.
- tutto il resto resta byte per byte (latin1 grezzo con dentro UTF-8 e cp1252).

Uso: scripts/anonimizza-coppermine.py
"""
import gzip
import hashlib
import hmac
import os
import re
import unicodedata
from pathlib import Path

RADICE = Path(__file__).resolve().parent.parent
GREZZO = RADICE / "grezzo" / "coppermine.tsv.gz"
USCITA = RADICE / "testdata" / "coppermine.sql.gz"

CHIAVE = os.urandom(32)  # mai salvata: la pseudonimizzazione non si può invertire
CONSONANTI, VOCALI = "bcdfglmnprstvz", "aeiou"
CHIAVI_PRIMARIE = {
    "cpg1410_pictures": "pid", "cpg1410_albums": "aid", "cpg1410_categories": "cid",
    "cpg1410_exif": "pid", "phpbb_users": "user_id",
}
# Ordine delle colonne nelle SELECT di scripts/estrai-coppermine.sh (diverso dallo schema).
SELEZIONE = {
    "cpg1410_pictures": "pid aid filepath filename filesize pwidth pheight hits ctime owner_id title caption keywords approved user1 user2 user3",
    "cpg1410_albums": "aid title description category pos visibility owner keyword",
    "cpg1410_categories": "cid owner_id name description pos parent",
    "cpg1410_exif": "pid exifData",
    "phpbb_users": "user_id username",
}
NUMERICI = re.compile(r"^(tiny|small|medium|big)?int|^decimal|^float|^double")
TOKEN = re.compile(r"&#?\w+;|[^\W\d_]+")


def parola_finta(parola: str) -> str:
    """Parola pronunciabile, stessa iniziale e lunghezza, scelta da HMAC."""
    if len(parola) == 1:
        return parola
    for prova in range(10):
        h = hmac.new(CHIAVE, f"{prova}:{parola.lower()}".encode(), hashlib.sha256).digest()
        iniziale = unicodedata.normalize("NFKD", parola[0]).encode("ascii", "ignore").decode() or "x"
        lettere = [iniziale.lower()]
        for i in range(1, len(parola)):
            serie = VOCALI if lettere[-1] in CONSONANTI else CONSONANTI
            lettere.append(serie[h[i % len(h)] % len(serie)])
        finta = "".join(lettere)
        if finta != parola.lower():
            break
    if parola.isupper():
        return finta.upper()
    if parola[0].isupper():
        return finta.capitalize()
    return finta


def cifre_finte(cifre: str) -> str:
    h = hmac.new(CHIAVE, cifre.encode(), hashlib.sha256).digest()
    return "".join(str(h[i] % 10) for i in range(len(cifre)))


def pseudonimo(grezzo: bytes, cifre: bool = False) -> bytes:
    """Le cifre si cambiano solo negli username: nell'Autore sono di solito anni."""
    try:
        testo, codifica = grezzo.decode("utf-8"), "utf-8"
    except UnicodeDecodeError:
        testo, codifica = grezzo.decode("cp1252", errors="replace"), "cp1252"
    sost = lambda m: m.group() if m.group().startswith("&") else parola_finta(m.group())
    testo = TOKEN.sub(sost, testo)
    if cifre:
        testo = re.sub(r"\d+", lambda m: cifre_finte(m.group()), testo)
    return testo.encode(codifica)


def exif_ridotto(grezzo: bytes) -> bytes:
    """exifData è un array PHP serializzato e piatto. Resta solo DateTimeOriginal, nello
    stesso formato. Coppermine non salva campi GPS (verificato sui dati del 18/10/2026)."""
    m = re.search(rb's:16:"DateTimeOriginal";s:(\d+):"', grezzo)
    if not m:
        return b"a:0:{}"
    valore = grezzo[m.end():m.end() + int(m.group(1))]
    return b'a:1:{s:16:"DateTimeOriginal";s:%d:"%s";}' % (len(valore), valore)


def valore_sql(b: bytes | str | None, tipo: str) -> str:
    if b is None:
        return "NULL"
    if NUMERICI.match(tipo):
        return b
    return "X'" + b.hex() + "'" if b else "''"


def senza_riservate(righe: dict[str, list[list[str | None]]]) -> dict[str, list[list[str | None]]]:
    """Tiene solo album con visibility 0, le Foto di quegli album e i loro EXIF. Una Foto di un
    album che non c'è si toglie: in dubbio non si pubblica. I valori sono esadecimali."""
    colonna = lambda tab, nome: SELEZIONE[tab].split().index(nome)
    aid, vis = colonna("cpg1410_albums", "aid"), colonna("cpg1410_albums", "visibility")
    album = [r for r in righe.get("cpg1410_albums", []) if int(r[vis], 16) == 0]
    pubblici = {int(r[aid], 16) for r in album}
    pid, foto_aid = colonna("cpg1410_pictures", "pid"), colonna("cpg1410_pictures", "aid")
    foto = [r for r in righe.get("cpg1410_pictures", []) if int(r[foto_aid], 16) in pubblici]
    tenute = {int(r[pid], 16) for r in foto}
    exif_pid = colonna("cpg1410_exif", "pid")
    exif = [r for r in righe.get("cpg1410_exif", []) if int(r[exif_pid], 16) in tenute]
    return righe | {"cpg1410_albums": album, "cpg1410_pictures": foto, "cpg1410_exif": exif}


def main() -> None:
    meta: dict[tuple[str, str], list[tuple[str, str, str, str]]] = {}
    righe: dict[str, list[list[bytes | None]]] = {}
    versione = ""
    with gzip.open(GREZZO, "rb") as f:
        for linea in f:
            campi = linea.rstrip(b"\n").split(b"\t")
            tag = campi[0].decode()
            if tag == "META":
                db, tab, col, tipo, coll, nullo, _ = (c.decode() for c in campi[1:])
                meta.setdefault((db, tab), []).append((col, tipo, coll, nullo))
            elif tag == "VERSION":
                versione = campi[1].decode()
            else:
                # Restano esadecimali: HEX() di un intero è il numero in base 16, non i suoi byte.
                righe.setdefault(tag, []).append([None if c == b"NULL" else c.decode() for c in campi[1:]])
    righe = senza_riservate(righe)

    out = [f"-- Dati di test anonimizzati da Coppermine (MySQL {versione}). Generato da scripts/anonimizza-coppermine.py.",
           "SET NAMES binary;"]
    for (db, tab), colonne in meta.items():
        nomi = [c[0] for c in colonne]
        out += [f"CREATE DATABASE IF NOT EXISTS `{db}`;", f"DROP TABLE IF EXISTS `{db}`.`{tab}`;"]
        defs = []
        for col, tipo, coll, nullo in colonne:
            cs = f" CHARACTER SET {coll.split('_')[0]} COLLATE {coll}" if coll else ""
            defs.append(f"`{col}` {tipo}{cs}{'' if nullo == 'YES' else ' NOT NULL'}")
        defs.append(f"PRIMARY KEY (`{CHIAVI_PRIMARIE[tab]}`)")
        out.append(f"CREATE TABLE `{db}`.`{tab}` (\n  " + ",\n  ".join(defs) + "\n);")
        tipi = [c[1] for c in colonne]
        tipo_di = dict(zip(nomi, tipi))
        dati = righe.get(tab, [])
        for riga in dati:
            r = {}
            for n, h in zip(SELEZIONE[tab].split(), riga):
                if h is None:
                    r[n] = None
                elif NUMERICI.match(tipo_di[n]):
                    v = int(h, 16)  # HEX(-1) = complemento a due su 64 bit
                    r[n] = str(v - 2**64 if v >= 2**63 else v)
                else:
                    r[n] = bytes.fromhex(h)
            if tab == "phpbb_users" and r["username"] is not None:
                r["username"] = pseudonimo(r["username"], cifre=True)
            if tab == "cpg1410_pictures" and r["user1"]:
                r["user1"] = pseudonimo(r["user1"])
            if tab == "cpg1410_exif" and r["exifData"] is not None:
                r["exifData"] = exif_ridotto(r["exifData"])
            riga[:] = [r[n] for n in nomi]
        for i in range(0, len(dati), 500):
            valori = ",\n".join("(" + ",".join(valore_sql(v, t) for v, t in zip(r, tipi)) + ")" for r in dati[i:i + 500])
            out.append(f"INSERT INTO `{db}`.`{tab}` (" + ",".join(f"`{n}`" for n in nomi) + f") VALUES\n{valori};")
    USCITA.parent.mkdir(exist_ok=True)
    with gzip.open(USCITA, "wt", encoding="ascii", compresslevel=9) as f:
        f.write("\n".join(out) + "\n")
    print(f"{USCITA}: " + ", ".join(f"{t} {len(r)}" for t, r in righe.items()))


if __name__ == "__main__":
    assert parola_finta("Rossi") == parola_finta("rossi").capitalize()
    assert parola_finta("M") == "M" and parola_finta("Mario")[0] == "M"
    assert pseudonimo(b"Mario &amp; ENEL").startswith(b"M") and b"&amp;" in pseudonimo(b"Mario &amp; ENEL")
    assert pseudonimo("Niccolò".encode()) .decode().isascii()
    # Album 1 pubblico, 2 riservato: della Foto 11 (album 2) e della 12 (album che non c'è) non resta nulla.
    prova = senza_riservate({
        "cpg1410_albums": [["1", "", "", "", "", "0", "", ""], ["2", "", "", "", "", "2", "", ""]],
        "cpg1410_pictures": [[f, a] + [""] * 15 for f, a in (("A", "1"), ("B", "2"), ("C", "7"))],
        "cpg1410_exif": [["A", ""], ["B", ""], ["C", ""]],
    })
    assert [r[0] for r in prova["cpg1410_albums"]] == ["1"]
    assert [r[0] for r in prova["cpg1410_pictures"]] == ["A"] and [r[0] for r in prova["cpg1410_exif"]] == ["A"]
    main()
