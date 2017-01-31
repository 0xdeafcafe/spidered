package models

import "net/url"

// PageEntity holds information about a specified crawlled page
type PageEntity struct {
	Path        string
	URL         *url.URL
	StatusCode  int
	ContentType string
	ContentSize int64
}
