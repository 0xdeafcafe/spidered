package models

import (
	"net/http"
	"net/url"
	"time"
)

// PageEntity holds information about a crawled page
type PageEntity struct {
	Path        string
	URL         *url.URL
	ContentType string
	CrawledAt   time.Time

	ResponseStatus   int
	ResponseHeaders  http.Header
	ResponseSize     int
	ResponseChecksum uint32
}
