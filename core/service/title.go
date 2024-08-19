package service

import (
	"bytes"
	"fmt"
	"github.com/projectdiscovery/stringsutil"
	"golang.org/x/net/html"
	"io"
	"regexp"
	"strings"
)

// 参考httpx的title识别逻辑
var (
	cutset        = "\n\t\v\f\r"
	reTitle       = regexp.MustCompile(`(?im)<\s*title.*>(.*?)<\s*/\s*title>`)
	reContentType = regexp.MustCompile(`(?im)\s*charset="(.*?)"|charset=(.*?)"\s*`)
)

func ExtractTitle(data []byte, raw string) (title string) {
	titleDom, err := getTitleWithDom(data)
	if err != nil {
		for _, match := range reTitle.FindAllString(raw, -1) {
			title = match
			break
		}
	} else {
		title = renderNode(titleDom)
	}

	title = html.UnescapeString(trimTitleTags(title))

	title = strings.TrimSpace(strings.Trim(title, cutset))
	title = stringsutil.ReplaceAll(title, "", "\n", "\t", "\v", "\f", "\r")

	return title
}

func getTitleWithDom(data []byte) (*html.Node, error) {
	var title *html.Node
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "title" {
			title = node
			return
		}
		for child := node.FirstChild; child != nil && title == nil; child = child.NextSibling {
			crawler(child)
		}
	}
	htmlDoc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	crawler(htmlDoc)
	if title != nil {
		return title, nil
	}
	return nil, fmt.Errorf("title not found")
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n) //nolint
	return buf.String()
}

func trimTitleTags(title string) string {
	titleBegin := strings.Index(title, ">")
	titleEnd := strings.Index(title, "</")
	if titleEnd < 0 || titleBegin < 0 {
		return title
	}
	return title[titleBegin+1 : titleEnd]
}
