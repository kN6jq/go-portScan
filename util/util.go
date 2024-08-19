// Package util -----------------------------
// @file      : util.go
// @author    : Xm17
// @contact   : https://github.com/kN6jq
// @time      : 2024/8/19 20:29
// -------------------------------------------
package util

import (
	"regexp"
)

// ExtractMatches 使用正则表达式从字符串中提取所有匹配项
func ExtractMatches(pattern, text string) ([][]string, error) {
	regxp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return regxp.FindAllStringSubmatch(text, -1), nil
}
