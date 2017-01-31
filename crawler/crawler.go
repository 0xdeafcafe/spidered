package crawler

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"net/url"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
	"gopkg.in/fatih/set.v0"

	"bytes"

	"time"

	"net/http"

	"github.com/0xdeafcafe/spidered/crawler/models"
)

var customUserAgent = "SpideredBot"

// Crawler holds information about a crawler
type Crawler struct {
	Domain          *url.URL
	CustomUserAgent string
	IgnoreRobots    bool
	RobotsData      *robotstxt.RobotsData
	SocketLimit     int
	Entities        map[string]*models.PageEntity
	Mutex           *sync.Mutex
}

// NewCrawler creates a new Crawler with the specified arguments
func NewCrawler(domain *url.URL, socketLimit int, ignoreRobots bool, userAgent string) (crawler *Crawler, err error) {
	crawler = &Crawler{
		Domain:          domain,
		CustomUserAgent: customUserAgent,
		IgnoreRobots:    ignoreRobots,
		SocketLimit:     socketLimit,
		Entities:        make(map[string]*models.PageEntity),
		Mutex:           &sync.Mutex{},
	}

	// Check the URL given is a root url, without a path
	if !IsRootURL(domain) {
		return nil, errors.New("provided url is not a root url")
	}

	// Set custom useragent
	if userAgent != "" {
		crawler.CustomUserAgent = userAgent
	}

	log.Infoln(fmt.Sprintf("Created a new Crawler - url: %s, socketLimit: %d, ignoreRobots: %t, customUserAgent: %s", domain.String(), socketLimit, ignoreRobots, crawler.CustomUserAgent))
	return crawler, nil
}

// Crawl starts crawling the specified domain.
func (crawler Crawler) Crawl() {
	var wg sync.WaitGroup
	completedURLs := set.New()

	// Create the socket limit
	socketLimit := make(chan int, crawler.SocketLimit)
	for i := 0; i < crawler.SocketLimit; i++ {
		socketLimit <- 1
	}

	// Check if the user wants to ignore the robots.txt
	if !crawler.IgnoreRobots {
		resp, err := MakeRequest("GET", crawler.Domain.String()+"robots.txt", crawler.CustomUserAgent)
		if err != nil {
			log.Fatalln(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			robots, err := robotstxt.FromResponse(resp)
			fmt.Println(resp)
			if err != nil {
				log.Fatalln(err)
				return
			}
			crawler.RobotsData = robots
		} else {
			log.Warnln(fmt.Sprintf("Unable to load robots for this domain. Response was %d", resp.StatusCode))
		}

	}

	// Increment the WaitGroup, crawl, and wait until we've finished crawling
	wg.Add(1)
	go crawlURL(&crawler, crawler.Domain, completedURLs, &wg, socketLimit)
	wg.Wait()
}

func crawlURL(crawler *Crawler, url *url.URL, completedURLs *set.Set, wg *sync.WaitGroup, socketLimit chan int) {
	defer wg.Done()
	urlStr := url.String()
	completedURLs.Add(urlStr)
	log.Infoln(fmt.Sprintf("Crawling new URL: %s", urlStr))

	// Wait until we have an opening to open this socket
	<-socketLimit
	resp, err := MakeRequest("GET", urlStr, crawler.CustomUserAgent)
	socketLimit <- 1
	if err != nil {
		log.Errorln(err)
		completedURLs.Remove(urlStr)
		return
	}
	defer resp.Body.Close()

	// Read Body into []byte
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return
	}

	// Create new page entity
	pageEntity := &models.PageEntity{
		URL:         url,
		Path:        url.Path,
		ContentType: resp.Header.Get("Content-Type"),
		CrawledAt:   time.Now(),

		ResponseHeaders:  resp.Header,
		ResponseStatus:   resp.StatusCode,
		ResponseSize:     len(body),
		ResponseChecksum: crc32.ChecksumIEEE(body),
	}

	// Create new reader for the tokenizer
	tokenReader := bytes.NewReader(body)
	tokenizer := html.NewTokenizer(tokenReader)
	for {
		tokenType := tokenizer.Next()
		switch {
		case tokenType == html.ErrorToken:
			// We've hit the end of the page - save page entity and return
			crawler.Mutex.Lock()
			crawler.Entities[urlStr] = pageEntity
			crawler.Mutex.Unlock()
			log.Infoln(fmt.Sprintf("URL crawling complete: %s", urlStr))
			return

		case tokenType == html.StartTagToken:
			token := tokenizer.Token()

			if token.Data != "a" {
				continue
			}

			href := GetAttribute(token, "href")
			if href == "" {
				continue
			}

			// Convert the URL and check if we want to crawl it
			url := ConvertToURL(href, crawler.Domain)
			if !IsRelevantURL(crawler.Domain, url) {
				log.Infoln(fmt.Sprintf("Skipping irrelevant URL: %s", url))
				continue
			}

			// Check if we're obeying our robot overlords
			if !crawler.IgnoreRobots && crawler.RobotsData != nil {
				allowURL := crawler.RobotsData.TestAgent(url.Path, crawler.CustomUserAgent)
				if !allowURL {
					log.Infoln(fmt.Sprintf("Skipping URL as per robot: %s", url))
					continue
				}
			}

			// If the URL has already been crawled, skip it
			if completedURLs.Has(url.String()) {
				log.Infoln(fmt.Sprintf("URL already crawled: %s", url))
				continue
			}

			// Increment the WaitGroup and crawl the URL
			wg.Add(1)
			go crawlURL(crawler, url, completedURLs, wg, socketLimit)
			break
		}
	}
}
