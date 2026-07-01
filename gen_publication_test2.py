#!/usr/bin/env python3
"""
Génère publication_test2.csv depuis Export_Liste des publications_*.csv.
Règles :
  - une seule ligne par p_ident (dernière occurrence comme base)
  - pf_abstract = dernier pf_file dont pf_type == "abstract"
  - pf_file     = dernier pf_file toutes catégories confondues
"""
import csv
import sys

SRC = "Export_Liste des publications_26062026.csv"
DST = "publication_test2.csv"


def main():
    with open(SRC, encoding="utf-8") as f:
        reader = csv.reader(f, delimiter=";")
        header_src = next(reader)

        idx_ident   = header_src.index("p_ident")
        idx_pf_type = header_src.index("pf_type")
        idx_pf_file = header_src.index("pf_file")

        # Collect rows grouped by ident, preserving file order
        ident_rows: dict[str, list] = {}
        ident_order: list[str] = []
        for row in reader:
            if len(row) <= idx_ident:
                continue
            ident = row[idx_ident]
            if ident not in ident_rows:
                ident_rows[ident] = []
                ident_order.append(ident)
            ident_rows[ident].append(row)

    # Output header: insert pf_abstract right after pf_file
    header_dst = header_src[: idx_pf_file + 1] + ["pf_abstract"] + header_src[idx_pf_file + 1 :]

    with open(DST, "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f, delimiter=";", quoting=csv.QUOTE_ALL)
        writer.writerow(header_dst)

        for ident in ident_order:
            rows = ident_rows[ident]
            base = list(rows[-1])  # dernière ligne comme base

            # Dernier pf_file où pf_type == "abstract"
            pf_abstract = ""
            for row in reversed(rows):
                pf_type = row[idx_pf_type] if len(row) > idx_pf_type else ""
                pf_file  = row[idx_pf_file]  if len(row) > idx_pf_file  else ""
                if pf_type == "abstract" and pf_file not in ("", "NULL"):
                    pf_abstract = pf_file
                    break

            # Dernier pf_file toutes catégories
            pf_file_final = ""
            for row in reversed(rows):
                pf_file = row[idx_pf_file] if len(row) > idx_pf_file else ""
                if pf_file not in ("", "NULL"):
                    pf_file_final = pf_file
                    break

            # Appliquer les valeurs calculées dans la ligne de base
            if len(base) > idx_pf_file:
                base[idx_pf_file] = pf_file_final

            out_row = base[: idx_pf_file + 1] + [pf_abstract] + base[idx_pf_file + 1 :]
            writer.writerow(out_row)

    print(f"Écrit {len(ident_order)} lignes dans {DST}")


if __name__ == "__main__":
    main()
