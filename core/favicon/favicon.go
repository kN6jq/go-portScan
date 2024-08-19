// Package favicon -----------------------------
// @file      : favicon.go
// @author    : Xm17
// @contact   : https://github.com/kN6jq
// @time      : 2024/8/19 20:28
// -------------------------------------------
package favicon

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/twmb/murmur3"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// GetFavicon 从 HTTP 响应中提取 favicon 路径，并生成其哈希值
func GetFavicon(httpBody, baseURL string) string {
	faviconPattern := `href="(.*?favicon\.[a-zA-Z]{2,4})"`
	faviconPaths, err := extractMatches(faviconPattern, httpBody)
	if err != nil {
		return "0"
	}

	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return "0"
	}
	baseURL = baseURLParsed.Scheme + "://" + baseURLParsed.Host

	var faviconPath string
	if len(faviconPaths) > 0 {
		fav := faviconPaths[0][1]
		switch {
		case fav[:2] == "//":
			faviconPath = "http:" + fav
		case fav[:4] == "http":
			faviconPath = fav
		default:
			faviconPath = baseURL + "/" + fav
		}
	} else {
		faviconPath = baseURL + "/favicon.ico"
	}

	return hashFavicon(faviconPath)
}

// extractMatches 使用正则表达式从文本中提取匹配项
func extractMatches(pattern, text string) ([][]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllStringSubmatch(text, -1), nil
}

// hashFavicon 获取 favicon 文件并计算其哈希值
func hashFavicon(url string) string {
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "0"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "0"
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "0"
	}

	return mmh3Hash32(encodeBase64(body))
}

// mmh3Hash32 计算给定字节数据的 MurmurHash3 32 位哈希值
func mmh3Hash32(data []byte) string {
	hash := murmur3.New32()
	if _, err := hash.Write(data); err != nil {
		return "0"
	}
	return fmt.Sprintf("%d", hash.Sum32())
}

// encodeBase64 将字节数据编码为标准 Base64 格式，并按照每行 76 个字符分隔
func encodeBase64(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	var buffer bytes.Buffer
	for i := 0; i < len(encoded); i++ {
		buffer.WriteByte(encoded[i])
		if (i+1)%76 == 0 {
			buffer.WriteByte('\n')
		}
	}
	return buffer.Bytes()
}
