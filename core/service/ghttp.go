package service

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kN6jq/go-portScan/core/favicon"
	"github.com/kN6jq/go-portScan/core/finger"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// HTTP defines the plain http scheme
	HTTP = "http"
	// HTTPS defines the secure http scheme
	HTTPS = "https"
	// HTTPorHTTPS defines the both http and https scheme
	HTTPorHTTPS = "http|https"
)

type ScanOptions struct {
	Methods                []string
	StoreResponseDirectory string
	RequestURI             string
	RequestBody            string
	VHost                  bool
	OutputTitle            bool
	OutputStatusCode       bool
	OutputLocation         bool
	OutputContentLength    bool
	StoreResponse          bool
	OutputServerHeader     bool
	OutputWebSocket        bool
	OutputWithNoColor      bool
	OutputMethod           bool
	ResponseInStdout       bool
	TLSProbe               bool
	CSPProbe               bool
	OutputContentType      bool
	Unsafe                 bool
	Pipeline               bool
	HTTP2Probe             bool
	OutputIP               bool
	OutputCName            bool
	OutputCDN              bool
	OutputResponseTime     bool
	PreferHTTPS            bool
	NoFallback             bool
}

func Analyze(protocol string, domain net.IP, port uint16, method string, scanopts *ScanOptions) Result {
	origProtocol := protocol
	if protocol == "http" {
		protocol = HTTP
	} else {
		protocol = HTTPS
	}
	retried := false
retry:
	URL := fmt.Sprintf("%s://%s", protocol, domain)
	if port > 0 {
		URL = fmt.Sprintf("%s://%s:%d", protocol, domain, port)
	}

	var client *http.Client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{
		Timeout:   time.Second * 10, //timeout
		Transport: tr,
	}

	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return Result{URL: URL, err: err}
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36")

	resp, err := client.Do(req)

	if err != nil {
		// https -> http相互转化
		if !retried && origProtocol == HTTPorHTTPS {
			if protocol == HTTPS {
				protocol = HTTP
			} else {
				protocol = HTTPS
			}
			retried = true
			goto retry
		}
		return Result{URL: URL, err: err}
	}

	var fullURL string

	if resp.StatusCode >= 0 {
		if port > 0 {
			fullURL = fmt.Sprintf("%s://%s:%d", protocol, domain, port)
		} else {
			fullURL = fmt.Sprintf("%s://%s", protocol, domain)
		}
	}

	builder := &strings.Builder{}
	builder.WriteString(fullURL)

	if scanopts.OutputStatusCode {
		builder.WriteString(" [")
		builder.WriteString(strconv.Itoa(resp.StatusCode))
		builder.WriteRune(']')
	}

	if scanopts.OutputContentLength {
		builder.WriteString(" [")
		builder.WriteString(strconv.FormatInt(resp.ContentLength, 10))
		builder.WriteRune(']')
	}

	if scanopts.OutputContentType {
		builder.WriteString(" [")
		builder.WriteString(resp.Header.Get("Content-Type"))
		builder.WriteRune(']')
	}

	defer resp.Body.Close()
	var titlex []string
	var fingerx string
	var faviconx string
	var respbody []byte
	body, err := ioutil.ReadAll(resp.Body)
	respbody, err = DecodeData(body, resp.Header)
	if err != nil {
		return Result{URL: URL, err: err}
	}
	if err == nil {
		//todo favicon 识别
		title1 := ExtractTitle(respbody, string(body)) // title识别
		if title1 != "" {
			titlex = append(titlex, title1)
		}
		finger1 := finger.ExtractFinger(string(body), title1, resp)
		if finger1 != "" {
			fingerx = finger1
		}
		favicon1 := favicon.GetFavicon(string(body), fullURL)
		if favicon1 != "" {
			faviconx = favicon1
		}
		if scanopts.OutputTitle {
			builder.WriteString(" [")
			builder.WriteString(strings.Join(titlex, "|"))
			builder.WriteRune(']')
		}
	}
	title := strings.Join(titlex, "|")

	serverHeader1 := resp.Header.Get("Server")
	serverHeader2 := resp.Header.Get("X-Powered-By")
	var serverHeaders []string
	if serverHeader1 != "" {
		serverHeaders = append(serverHeaders, serverHeader1)
	}
	if serverHeader2 != "" {
		serverHeaders = append(serverHeaders, serverHeader2)
	}
	serverHeader := strings.Join(serverHeaders, "|")

	if scanopts.OutputServerHeader {
		builder.WriteString(fmt.Sprintf(" [%s]", serverHeader))
	}

	// web socket
	isWebSocket := resp.StatusCode == 101
	if scanopts.OutputWebSocket && isWebSocket {
		builder.WriteString(" [websocket]")
	}

	return Result{
		URL:           fullURL,
		ContentLength: len(body),
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		Title:         title,
		WebServer:     serverHeader,
		str:           builder.String(),
		Finger:        fingerx,
		Favicon:       faviconx,
	}
}

// Result of a scan
type Result struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	WebServer     string `json:"webserver"`
	ContentType   string `json:"content-type,omitempty"`
	ContentLength int    `json:"content-length"`
	StatusCode    int    `json:"status-code"`
	Finger        string `json:"finger"`
	Favicon       string `json:favicon`
	err           error
	str           string
}

// JSON the result
func (r *Result) JSON() string {
	if js, err := json.Marshal(r); err == nil {
		return string(js)
	}

	return ""
}

func GetHttpTitle(target net.IP, proc string, port uint16) Result {
	var scanopts = new(ScanOptions)
	scanopts.OutputTitle = true
	scanopts.OutputServerHeader = true
	result := Analyze(proc, target, port, "GET", scanopts)
	return result
}

func (r *Result) ToString() string {

	builder := &bytes.Buffer{}
	if r.err == nil {
		builder.WriteString("[")
		builder.WriteString(ToString(r.StatusCode))
		builder.WriteString("] ")
		if r.WebServer != "" {
			builder.WriteString("[")
			builder.WriteString(r.WebServer)
			builder.WriteString("] ")
		}
		if r.Title != "" {
			builder.WriteString("[")
			builder.WriteString(r.Title)
			builder.WriteString("] ")
		}
	}

	return builder.String()
}

func hostsFrom(ss []string) []string {
	for i, s := range ss {
		u, _ := url.Parse(s)
		if host := u.Hostname(); host != "" {
			ss[i] = host
		}
	}
	return ss
}

type hostinfo struct {
	Host  net.IP
	Port  uint16
	Certs []*x509.Certificate
}

func (h *hostinfo) getCerts(timeout time.Duration) error {
	//log.Printf("connecting to %s:%d", h.Host, h.Port)
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(
		dialer,
		"tcp",
		//h.Host+":"+strconv.Itoa(h.Port),
		h.Host.String()+":"+strconv.Itoa(int(h.Port)),
		&tls.Config{
			InsecureSkipVerify: true,
		})
	if err != nil {
		return err
	}

	defer conn.Close()

	if err := conn.Handshake(); err != nil {
		return err
	}

	pc := conn.ConnectionState().PeerCertificates
	h.Certs = make([]*x509.Certificate, 0, len(pc))
	for _, cert := range pc {
		if cert.IsCA {
			continue
		}
		h.Certs = append(h.Certs, cert)
	}

	return nil
}

func CertInfo(host net.IP, port uint16, timeout time.Duration) (commonName string, dnsNames []string, err error) {
	//port_int, err := strconv.Atoi(port)
	if err != nil {
		return commonName, dnsNames, err
	}
	info := hostinfo{Host: host, Port: port}
	err = info.getCerts(timeout)
	if err != nil {
		return commonName, dnsNames, err
	}
	for _, cert := range info.Certs {
		if cert != nil && cert.Subject.CommonName != "" {
			return cert.Subject.CommonName, cert.DNSNames, err
		}
	}
	return commonName, dnsNames, errors.New("not found")
}

func GetCert(domain net.IP, port uint16) (string, error) {
	var CN string
	var DN []string
	var ret string
	var err error
	if port > 0 {
		CN, DN, err = CertInfo(domain, port, 5*time.Second)
	} else {
		CN, DN, err = CertInfo(domain, 443, 5*time.Second)
	}
	ret = "CommonName:" + CN + "; "
	if len(DN) > 0 {
		ret = ret + "DNSName:"
		ret = ret + DN[0]
	}
	return ret, err
}
