package domain

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"sqldb-ws/domain/domain_service/permission"
	ds "sqldb-ws/domain/schema/database_resources"
	"sqldb-ws/domain/utils"
	"strings"
)

var params = map[string][]*utils.Params{} // SHOULD add comparision. in params

func addParams(tableName string, p utils.Params) (int, utils.Params) {
	if i := checkParamsAlreadyExists(tableName, p); i != -1 {
		return i, *params[tableName][i]
	}
	if params[tableName] == nil {
		params[tableName] = []*utils.Params{}
	}
	i := len(params[tableName])
	params[tableName] = append(params[tableName], &p)
	return i, p
}

func checkParamsAlreadyExists(tableName string, p utils.Params) int {
	if params[tableName] == nil {
		return -1
	}
	for i, pp := range params[tableName] {
		if pp.Compare(p) {
			return i
		}
	}
	return -1
}

var cache = map[string]map[string]map[int]*string{} // userID -> tablename -> params index -> data

func AddInCache(userID string, tableName string, method utils.Method, params utils.Params, res utils.Results) {
	if method != utils.SELECT {
		return
	}
	i, _ := addParams(tableName, params)
	if cache[userID] == nil {
		cache[userID] = map[string]map[int]*string{}
	}
	if cache[userID][tableName] == nil {
		cache[userID][tableName] = map[int]*string{}
	}
	str, err := CompressMap(res)
	if err == nil {
		cache[userID][tableName][i] = &str
	}

}

func deleteInCache(userID string, tableName string) {
	if cache[userID] == nil {
		return
	}
	if strings.Contains(tableName, "user") || strings.Contains(tableName, "role") || strings.Contains(tableName, "permission") || strings.Contains(tableName, "entity") {
		cache = map[string]map[string]map[int]*string{}
		delete(permission.CachePerms, userID)
		return
	}
	if cache[userID][tableName] == nil {
		return
	}
	delete(cache[userID], ds.DBView.Name)
	delete(cache[userID], tableName)
}

func GetInCache(userID string, tableName string, method utils.Method, params utils.Params) (bool, utils.Results) {
	if method != utils.SELECT {
		deleteInCache(userID, tableName)
		return false, utils.Results{}
	}
	if cache[userID] == nil || cache[userID][tableName] == nil {
		return false, utils.Results{}
	}
	i, _ := addParams(tableName, params)
	if cache[userID][tableName][i] == nil {
		return false, utils.Results{}
	}
	dp, err := DecompressMap(*cache[userID][tableName][i])
	if err != nil {
		return false, utils.Results{}
	}
	return true, dp
}

func removeNonLatin1(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if r <= 255 { // Latin-1
			result = append(result, r)
		}
	}
	return string(result)
}

func CompressMap(m utils.Results) (string, error) {
	// 1. Convert map to JSON
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	// 2. Compress with gzip
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Name = removeNonLatin1(gz.Name)
	if _, err := gz.Write(jsonBytes); err != nil {
		return "", err
	}
	gz.Close()

	// 3. Encode to base64
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// --- DECOMPRESS MAP ---
func DecompressMap(encoded string) (utils.Results, error) {
	// 1. Base64 decode
	compressedData, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	// 2. Gzip decompress
	gr, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if _, err := io.Copy(&out, gr); err != nil {
		return nil, err
	}
	gr.Close()

	// 3. Unmarshal JSON back to map
	var m utils.Results
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		return nil, err
	}
	return m, nil
}
