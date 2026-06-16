#!/bin/bash

INPUT="publication_test4.csv"
OUTPUT="publication_test.csv"

awk -F';' '
NR==1 {
    for (i=1; i<=NF; i++) {
        if ($i == "\"p_actif\"") {
            col=i
            break
        }
    }
    print
    next
}

$col == "\"1\""
' "$INPUT" > "$OUTPUT"