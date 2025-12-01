package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/unidoc/unioffice/document"
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
	filePath := filename
	if !strings.Contains(filePath, "/mnt/files/") {
		filePath = "/mnt/files/" + filename
	}
	text, err := readFileAsText(filePath)
	if err != nil {
		fmt.Println("can't read file as text", filePath, err)
		return false
	}

	if strings.Contains(strings.ToLower(text), strings.ToLower(searchTerm)) {
		return true
	} else {
		return false
	}
}

func readFileAsText(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".txt":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case ".docx":
		doc, err := document.Open(path)
		if err != nil {
			return "", err
		}
		var text string
		for _, para := range doc.Paragraphs() {
			for _, run := range para.Runs() {
				text += run.Text()
			}
		}
		return text, nil

	case ".pdf":
		f, r, err := pdf.Open(path)
		defer f.Close()
		if err != nil {
			return "", err
		}
		var text string
		totalPage := r.NumPage()
		for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
			p := r.Page(pageIndex)
			if p.V.IsNull() {
				continue
			}
			text, err := p.GetPlainText(nil)
			if err != nil {
				return "", err
			}
			text += text
		}
		return text, nil
	default:
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
}
