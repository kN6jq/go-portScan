package service

import (
	"bytes"
	"fmt"
	"github.com/projectdiscovery/stringsutil"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Decodegbk converts GBK to UTF-8
func Decodegbk(s []byte) ([]byte, error) {
	I := bytes.NewReader(s)
	O := transform.NewReader(I, simplifiedchinese.GBK.NewDecoder())
	d, e := io.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// ExtractTitle from a response
func DecodeData(data []byte, headers http.Header) ([]byte, error) {
	// Non UTF-8
	if contentTypes, ok := headers["Content-Type"]; ok {
		contentType := strings.ToLower(strings.Join(contentTypes, ";"))

		switch {
		case stringsutil.ContainsAny(contentType, "charset=gb2312", "charset=gbk"):
			return Decodegbk([]byte(data))
		}

		// Content-Type from head tag
		var match = reContentType.FindSubmatch(data)
		var mcontentType = ""
		if len(match) != 0 {
			for i, v := range match {
				if string(v) != "" && i != 0 {
					mcontentType = string(v)
				}
			}
			mcontentType = strings.ToLower(mcontentType)
		}
		switch {
		case stringsutil.ContainsAny(mcontentType, "gb2312", "gbk"):
			return Decodegbk(data)
		}
	}

	// return as is
	return data, nil
}

// ToString converts an interface to string in a quick way
func ToString(data interface{}) string {
	switch s := data.(type) {
	case nil:
		return ""
	case string:
		return s
	case bool:
		return strconv.FormatBool(s)
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(s), 'f', -1, 32)
	case int:
		return strconv.Itoa(s)
	case int64:
		return strconv.FormatInt(s, 10)
	case int32:
		return strconv.Itoa(int(s))
	case int16:
		return strconv.FormatInt(int64(s), 10)
	case int8:
		return strconv.FormatInt(int64(s), 10)
	case uint:
		return strconv.FormatUint(uint64(s), 10)
	case uint64:
		return strconv.FormatUint(s, 10)
	case uint32:
		return strconv.FormatUint(uint64(s), 10)
	case uint16:
		return strconv.FormatUint(uint64(s), 10)
	case uint8:
		return strconv.FormatUint(uint64(s), 10)
	case []byte:
		return string(s)
	case fmt.Stringer:
		return s.String()
	case error:
		return s.Error()
	default:
		return fmt.Sprintf("%v", data)
	}
}
