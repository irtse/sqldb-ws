package main

import (
	"compress/gzip"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	csvPath := flag.String("csv", "publication_test.csv", "chemin vers le fichier CSV")
	prefix := flag.String("prefix", "/dynfiles/dynfiles", "répertoire source des fichiers")
	storage := flag.String("storage", "./files2", "répertoire de destination (stockage compressé)")
	flag.Parse()

	if err := os.MkdirAll(*storage, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "création storage: %v\n", err)
		os.Exit(1)
	}

	paths, err := extractUniquePaths(*csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lecture CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%d chemins uniques à traiter\n", len(paths))
	ok, fail := 0, 0
	for _, relPath := range paths {
		srcPath := filepath.Join(*prefix, relPath)
		if err := compressAndSave(srcPath, *storage); err != nil {
			fmt.Fprintf(os.Stderr, "ECHEC %s: %v\n", relPath, err)
			fail++
		} else {
			fmt.Printf("OK    %s\n", relPath)
			ok++
		}
	}
	fmt.Printf("\n%d réussis / %d échoués\n", ok, fail)
}

func extractUniquePaths(csvPath string) ([]string, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("lecture en-tête: %w", err)
	}

	fileCol, abstractCol := -1, -1
	for i, h := range header {
		switch h {
		case "pf_file":
			fileCol = i
		case "pf_abstract":
			abstractCol = i
		}
	}
	if fileCol == -1 {
		return nil, fmt.Errorf("colonne pf_file introuvable")
	}
	if abstractCol == -1 {
		return nil, fmt.Errorf("colonne pf_abstract introuvable")
	}

	seen := map[string]bool{}
	var paths []string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "ligne ignorée: %v\n", err)
			continue
		}
		for _, idx := range []int{fileCol, abstractCol} {
			if idx >= len(row) {
				continue
			}
			val := row[idx]
			if val == "" || val == "NULL" {
				continue
			}
			if !seen[val] {
				seen[val] = true
				paths = append(paths, val)
			}
		}
	}
	return paths, nil
}

// compressAndSave reproduit la logique de domain_service.Uploader.upload :
// supprime les versions précédentes, compresse en gzip et sauvegarde dans storage.
func compressAndSave(srcPath, storage string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	fileName := filepath.Base(srcPath)

	deleteExisting(storage, fileName)

	dstPath := filepath.Join(storage, strings.TrimSpace(fileName)+".gz")
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("création destination: %w", err)
	}
	defer dst.Close()

	gw := gzip.NewWriter(dst)
	gw.Name = strings.TrimSpace(fileName)
	defer gw.Close()

	if _, err := io.Copy(gw, src); err != nil {
		return fmt.Errorf("compression: %w", err)
	}
	return nil
}

// deleteExisting supprime toute version existante du même fichier (même logique que deleteBeforeUpload).
func deleteExisting(storage, fileName string) {
	baseName := strings.Split(fileName, ".")[0]
	pattern := `^` + regexp.QuoteMeta(baseName) + `.*`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return
	}
	filepath.Walk(storage, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if re.MatchString(info.Name()) {
			os.Remove(path)
		}
		return nil
	})
}
