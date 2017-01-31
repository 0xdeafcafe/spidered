package crawler

import (
	"net/url"

	"net/http"

	"strings"

	"golang.org/x/net/html"
)

// ConvertToURL ..
func ConvertToURL(path string, domain *url.URL) *url.URL {
	pathURL, _ := url.Parse(path)

	if pathURL.Scheme == "" {
		pathURL.Scheme = domain.Scheme
	}

	if pathURL.Host == "" {
		pathURL.Host = domain.Host
	}

	if strings.ContainsRune(pathURL.Path, '#') {
		pathURL.Path = strings.Split(pathURL.Path, "#")[0]
	}

	return pathURL
}

// IsSatisfiedURL ..
func IsSatisfiedURL(baseDomain *url.URL, crawledDomain *url.URL) bool {
	satisfied := true

	if baseDomain.Host != crawledDomain.Host {
		satisfied = false
	}

	if crawledDomain.Host == "" {
		satisfied = false
	}

	if crawledDomain.Scheme != "http" && crawledDomain.Scheme != "https" {
		satisfied = false
	}

	return satisfied
}

// GetAttribute ..
func GetAttribute(token html.Token, attrName string) string {
	for _, a := range token.Attr {
		if a.Key == attrName {
			return a.Val
		}
	}

	return ""
}

// MakeRequest ..
func MakeRequest(method, url, userAgent string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
