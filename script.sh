awk -F';' '
NR==1 {
    for (i=1; i<=NF; i++) {
        if ($i == "p_actif") {
            col = i
            break
        }
    }
    print
    next
}
$col == 1
' input.csv > output.csv
