package utils

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func BuildPath(tableName string, rows string, extra ...string) string {
	path := fmt.Sprintf("/%s/%s?rows=%v", MAIN_PREFIX, tableName, rows)
	for _, ext := range extra {
		path += "&" + ext
	}
	return path
}

func PrepareEnum(enum string) string {
	if !strings.Contains(enum, "enum") {
		return enum
	}
	return TransformType(enum)
}

func TransformType(enum string) string {
	e := strings.Replace(ToString(enum), " ", "", -1)
	e = strings.Replace(e, "'", "", -1)
	e = strings.Replace(e, "(", "__", -1)
	e = strings.Replace(e, ",", "_", -1)
	e = strings.Replace(e, ")", "", -1)
	return strings.ToLower(e)
}

func ToMap(who interface{}) map[string]interface{} {
	if who != nil && reflect.TypeOf(who).Kind() == reflect.Map {
		return who.(map[string]interface{})
	}
	return map[string]interface{}{}
}

func ToListAnonymized(who []string) []interface{} {
	i := []interface{}{}
	if who != nil && reflect.TypeOf(who).Kind() == reflect.Slice {
		for _, w := range who {
			i = append(i, w)
		}
		return i
	}
	return i
}

func ToList(who interface{}) []interface{} {
	if who == nil {
		return []interface{}{}
	}
	if reflect.TypeOf(who).Kind() == reflect.Slice {
		return who.([]interface{})
	}
	return []interface{}{}
}

func ToFloat64(who interface{}) float64 {
	if who == nil {
		return 0
	}
	i, err := strconv.ParseFloat(fmt.Sprintf("%v", who), 64)
	if err != nil {
		return 0
	}
	return float64(i)
}

func ToInt64(who interface{}) int64 {
	if who == nil {
		return 0
	}
	i, err := strconv.Atoi(fmt.Sprintf("%v", who))
	if err != nil {
		return 0
	}
	return int64(i)
}

func ToString(who interface{}) string {
	if who == nil {
		return ""
	}
	return fmt.Sprintf("%v", who)
}

func Compare(who interface{}, what interface{}) bool {
	return who != nil && fmt.Sprintf("%v", who) == fmt.Sprintf("%v", what)
}

func Translate(str string) string {
	url := "https://libretranslate.com/translate"
	target := os.Getenv("LANG")
	if target == "" {
		target = "fr"
	}

	data := map[string]string{
		"q":      str,
		"source": "en",
		"target": target,
		"format": "text",
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if body, err := io.ReadAll(resp.Body); err == nil {
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		return GetString(result, "translatedText")
	}
	return str
}

func SearchInFile(filename string, searchTerm string) bool {
	if !strings.Contains(filename, "/mnt/files/") {
		filename = "/mnt/files/" + filename
	}
	fmt.Println(filename)
	text, err := readFileAsText(filename)
	fmt.Println("ERROR", err)
	if err != nil {
		return false
	}
	fmt.Println(text, searchTerm, strings.Contains(strings.ToLower(text), strings.ToLower(searchTerm)))
	return strings.Contains(strings.ToLower(text), strings.ToLower(searchTerm))
}

func readFileAsText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if strings.Contains(path, ".txt") || strings.Contains(path, ".md") || strings.Contains(path, ".rst") {
		return readTXT(path)
	} else if strings.Contains(path, ".tex") {
		return readTEX(path)
	} else if strings.Contains(path, ".rtf") || strings.Contains(path, ".rtx") {
		return readRTF(path)
	} else if strings.Contains(path, ".docx") {
		return readDOCX(path)
	} else if strings.Contains(path, ".odt") {
		return readODT(path)
	} else if strings.Contains(path, ".fodt") {
		return readFODT(path)
	} else if strings.Contains(path, ".abw") {
		return readABW(path)
	} else {
		return "", fmt.Errorf("unsupported format: %s", ext)
	}
}

func readTXT(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

func readTEX(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Remove LaTeX commands \command{...}
	re := regexp.MustCompile(`\\[a-zA-Z]+\{.*?\}`)
	cleaned := re.ReplaceAllString(string(data), "")
	return cleaned, nil
}

func readRTF(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	text := string(data)

	// Remove RTF commands like \b, \par, \f1, \fs24
	text = regexp.MustCompile(`\\[a-zA-Z]+\d*`).ReplaceAllString(text, "")
	// Remove braces
	text = strings.ReplaceAll(text, "{", "")
	text = strings.ReplaceAll(text, "}", "")

	return text, nil
}

func readDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var docXML []byte

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ = io.ReadAll(rc)
			rc.Close()
		}
	}

	if docXML == nil {
		return "", fmt.Errorf("document.xml not found")
	}

	type Text struct {
		Text string `xml:",chardata"`
	}

	type Node struct {
		Runs []Text `xml:"t"`
	}

	type Document struct {
		Body struct {
			Paragraphs []Node `xml:"p"`
		} `xml:"body"`
	}

	var doc Document
	xml.Unmarshal(docXML, &doc)

	var sb strings.Builder
	for _, p := range doc.Body.Paragraphs {
		for _, t := range p.Runs {
			sb.WriteString(t.Text)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func readODT(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var xmlData []byte
	for _, f := range r.File {
		if f.Name == "content.xml" {
			rc, _ := f.Open()
			xmlData, _ = io.ReadAll(rc)
			rc.Close()
		}
	}

	if xmlData == nil {
		return "", fmt.Errorf("content.xml not found")
	}

	type P struct {
		Text string `xml:",chardata"`
	}

	type Content struct {
		Paragraphs []P `xml:"body>text>p"`
	}

	var doc Content
	xml.Unmarshal(xmlData, &doc)

	var sb strings.Builder
	for _, p := range doc.Paragraphs {
		sb.WriteString(p.Text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func readFODT(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	type P struct {
		Text string `xml:",chardata"`
	}

	type Doc struct {
		Paragraphs []P `xml:"body>text>p"`
	}

	var d Doc
	xml.Unmarshal(data, &d)

	var sb strings.Builder
	for _, p := range d.Paragraphs {
		sb.WriteString(p.Text)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func readABW(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	type P struct {
		Text string `xml:",chardata"`
	}

	type Doc struct {
		Paragraphs []P `xml:"body>p"`
	}

	var d Doc
	xml.Unmarshal(data, &d)

	var sb strings.Builder
	for _, p := range d.Paragraphs {
		sb.WriteString(p.Text)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func UncompressGzip(uncompressedPath string) (string, error) {
	// Ensure the file exists
	inFile, err := os.Open(fmt.Sprintf("%v.gz", strings.Trim(uncompressedPath, " ")))
	if err != nil {
		return "", fmt.Errorf("failed to open gzip file: %w", err)
	}
	defer inFile.Close()
	// Create a gzip reader
	gzipReader, err := gzip.NewReader(inFile)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	// Create destination file
	outFile, err := os.Create(uncompressedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create uncompressed file: %w", err)
	}
	defer outFile.Close()

	// Copy data from gzip -> destination file
	if _, err := io.Copy(outFile, gzipReader); err != nil {
		return "", fmt.Errorf("failed to decompress: %w", err)
	}

	return uncompressedPath, nil
}

func DeleteUncompressed(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete temp file: %w", err)
	}
	return nil
}
