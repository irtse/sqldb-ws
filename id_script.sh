#!/bin/bash

CSV="input.csv"
OUTPUT="missing_project.csv"

awk -F';' -v ids="$*" '
BEGIN {
    split(ids, a, " ")
    for (i in a)
        keep[a[i]]
}
FNR==1 {
    for (i = 1; i <= NF; i++)
        if ($i == "p_number")
            col = i
    print
    next
}
($col in keep)
' "$CSV" > "$OUTPUT"