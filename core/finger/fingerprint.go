package finger

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Packjson struct {
	Fingerprint []Fingerprint
}

type Fingerprint struct {
	Cms      string
	Method   string
	Location string
	Keyword  []string
}

//go:embed finger.json
var eHoleFinger string

var (
	Webfingerprint *Packjson
)

func init() {
	if err := LoadWebfingerprint(); err != nil {
		fmt.Println("Error loading Webfingerprint:", err)
		os.Exit(1)
	}
}

func LoadWebfingerprint() error {
	var config Packjson
	if err := json.Unmarshal([]byte(eHoleFinger), &config); err != nil {
		return err
	}
	Webfingerprint = &config
	return nil
}

func GetWebfingerprint() *Packjson {
	return Webfingerprint
}

func ExtractFinger(body, title string, r *http.Response) string {
	fingerprintData := GetWebfingerprint() // 采用更具描述性的变量名
	if fingerprintData == nil {
		fmt.Println("Fingerprint data is not loaded")
		return ""
	}

	headers := mapToJson(r.Header)
	var cms []string

	for _, finp := range fingerprintData.Fingerprint {
		source := getSourceContent(finp.Location, body, title, headers)
		if checkMethod(source, finp.Method, finp.Keyword) {
			cms = append(cms, finp.Cms)
		}
	}

	cms = removeDuplicatesAndEmpty(cms)
	return strings.Join(cms, ",")
}

func getSourceContent(location, body, title, headers string) string {
	switch location {
	case "body":
		return body
	case "header":
		return headers
	case "title":
		return title
	default:
		return ""
	}
}

func checkMethod(source, method string, keywords []string) bool {
	switch method {
	case "keyword":
		return allKeywordsMatch(source, keywords)
	case "regular":
		return allRegularsMatch(source, keywords)
	case "faviconhash":
		// 实现 faviconhash 的处理逻辑
		return false
	default:
		return false
	}
}

func allKeywordsMatch(source string, keywords []string) bool {
	for _, k := range keywords {
		if !strings.Contains(source, k) {
			return false
		}
	}
	return true
}

func allRegularsMatch(source string, keywords []string) bool {
	for _, k := range keywords {
		re, err := regexp.Compile(k)
		if err != nil {
			fmt.Println("Error compiling regex:", err)
			return false
		}
		if !re.MatchString(source) {
			return false
		}
	}
	return true
}

func mapToJson(param map[string][]string) string {
	dataType, err := json.Marshal(param)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return ""
	}
	return string(dataType)
}

func removeDuplicatesAndEmpty(a []string) []string {
	set := make(map[string]struct{})
	for _, item := range a {
		if len(item) > 0 {
			set[item] = struct{}{}
		}
	}
	var ret []string
	for key := range set {
		ret = append(ret, key)
	}
	return ret
}
