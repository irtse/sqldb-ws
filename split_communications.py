#!/usr/bin/env python3
"""
Split publication_test.csv (p_type = communication) into:
  - communication_peer_reviewed.csv
  - communication_not_peer_reviewed.csv

Classification rules (priority order):
  1. p_avec_abstract = "1"                                         → peer_reviewed (overrides everything)
  2. RCoC acronym (coc.txt) in p_nom_conference or p_nom_journal   → peer_reviewed (prime sur discriminants)
  3. No conference name                                             → not_peer_reviewed
  4. Discriminant term in titre / nom_conference / nom_journal      → not_peer_reviewed
     (Salon, Séminaire, Workshop, Technique(s) in conf/journal, Festival, Atelier technique,
      Ad'OCC, Table ronde, Journée technique, Invited talk)
  5. Incriminant term (Symposium, Brevets, Excellence scientifique reconnue) → peer_reviewed
  6. Default                                                        → not_peer_reviewed

  Note: RCoC check is limited to p_nom_conference + p_nom_journal (not p_titre) to avoid
  false positives where common words like "models" appear in scientific paper titles.
"""

import csv
import re
import sys
from pathlib import Path

BASE = Path(__file__).parent
INPUT_CSV  = BASE / "publication_test2.csv"
COC_FILE   = BASE / "coc.txt"
OUT_PR     = BASE / "communication_peer_reviewed.csv"
OUT_NPR    = BASE / "communication_not_peer_reviewed.csv"

DISCRIMINANT_TERMS = [
    "salon",
    "séminaire",
    "seminaire",
    "workshop",
    "festival",
    "atelier technique",
    "ad'occ",
    "table ronde",
    "journée technique",
    "journee technique",
    "invited talk",
]

INCRIMINANT_TERMS = [
    "symposium",
    "brevets",
    "excellence scientifique reconnue",
]


def load_rcoc(path: Path) -> list[str]:
    return [line.strip() for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def is_empty_conf(value: str) -> bool:
    v = value.strip().upper()
    return v in ("", "NULL")


def matches_rcoc(text: str, rcoc: list[str], min_len: int = 1) -> bool:
    # Word-boundary at start only: catches ERTS22, ECCM21, ICML2025…
    # but avoids AB matching inside words like "stabilité", "abstracts", etc.
    for acronym in rcoc:
        if len(acronym) >= min_len and re.search(r"\b" + re.escape(acronym), text, re.IGNORECASE):
            return True
    return False


def classify(row: list[str], rcoc: list[str]) -> str:
    p_avec_abstract  = row[40]
    p_titre          = row[5]
    p_nom_conference = row[8]
    p_nom_journal    = row[14]

    # Rule 1: abstract flag
    if p_avec_abstract == "1":
        return "peer_reviewed"

    searchable   = (p_titre + " " + p_nom_conference + " " + p_nom_journal).lower()
    conf_journal = (p_nom_conference + " " + p_nom_journal).lower()

    p_lien_conference = row[13]
    pf_file           = row[47]
    pf_abstract       = row[48]

    # Rule 2: RCoC acronym in conf, journal or conf link → peer_reviewed (overrides discriminants)
    if matches_rcoc(p_nom_conference + " " + p_nom_journal + " " + p_lien_conference, rcoc):
        return "peer_reviewed"
    # File names: require acronym ≥ 3 chars to avoid "AB" matching "Abstract" in filenames
    if matches_rcoc(pf_file + " " + pf_abstract, rcoc, min_len=3):
        return "peer_reviewed"

    # Rule 3: no conference name → not peer reviewed
    if is_empty_conf(p_nom_conference):
        return "not_peer_reviewed"

    # Rule 4: discriminant terms
    # "Technique(s)" checked only in conf/journal to avoid false positives in paper titles
    if re.search(r"\btechniques?\b", conf_journal):
        return "not_peer_reviewed"
    for term in DISCRIMINANT_TERMS:
        if term in searchable:
            return "not_peer_reviewed"

    # Rule 4: incriminant terms
    for term in INCRIMINANT_TERMS:
        if term in searchable:
            return "peer_reviewed"

    # Default
    return "not_peer_reviewed"


def main():
    rcoc = load_rcoc(COC_FILE)
    print(f"RCoC acronyms loaded: {len(rcoc)}")

    rows: list[list[str]] = []
    with open(INPUT_CSV, newline="", encoding="utf-8") as f:
        reader = csv.reader(f, delimiter=";", quotechar='"')
        headers = next(reader)
        for row in reader:
            rows.append(row)

    comms = [r for r in rows if len(r) > 4 and r[4] == "communication"]
    print(f"Total communications: {len(comms)}")

    peer, not_peer = [], []
    reasons: dict[str, list] = {}

    for row in comms:
        result = classify(row, rcoc)
        if result == "peer_reviewed":
            peer.append(row)
        else:
            not_peer.append(row)

    print(f"  → peer_reviewed:     {len(peer)}")
    print(f"  → not_peer_reviewed: {len(not_peer)}")

    def write_csv(path: Path, data: list[list[str]]):
        with open(path, "w", newline="", encoding="utf-8") as f:
            writer = csv.writer(f, delimiter=";", quotechar='"', quoting=csv.QUOTE_ALL)
            writer.writerow(headers)
            writer.writerows(data)
        print(f"Written: {path}  ({len(data)} rows)")

    write_csv(OUT_PR, peer)
    write_csv(OUT_NPR, not_peer)


if __name__ == "__main__":
    main()
