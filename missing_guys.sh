#!/bin/bash

FILE=${1:-"./user_test.csv"}
OUTPUT=${1:-"./rapport_matricules.csv"}

echo "Matricule,Doublon,Manquant" > "$OUTPUT"

awk -F',' '
NR > 1 {
    gsub(/^0+/, "", $2)
    if ($2 != "") {
        print $2
    }
}
' "$FILE" | sort -n | awk -v out="$OUTPUT" '

NR == 1 {
    prev = $1
    next
}

{
    current = $1

    # Doublon
    if (current == prev) {
        printf "%d,oui,non\n", current >> out
        doublons++
        next
    }

    # Manquants
    for (i = prev + 1; i < current; i++) {
        printf "%d,non,oui\n", i >> out
        manquants++
    }

    prev = current
}

END {
    printf "\nRésumé :\n"
    printf "Total manquants : %d\n", manquants
    printf "Total doublons  : %d\n", doublons
}
'
