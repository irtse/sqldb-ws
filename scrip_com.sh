#!/bin/bash

CSV="./publication_test4.csv"
OUTPUT="communications.csv"

awk -F';' '
FNR == 1 {
    for (i = 1; i <= NF; i++) {
        if ($i == "\"p_type\"")
            ptype_col = i
    }
    print
    next
}
$ptype_col == "\"communication\""
' "$CSV" > "$OUTPUT"