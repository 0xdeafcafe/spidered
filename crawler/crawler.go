package crawler

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/temoto/robotstxt"
	"golang.org/x/net/html"
	"gopkg.in/fatih/set.v0"

	"github.com/0xdeafcafe/spidered/crawler/models"
)

var customUserAgent = "SpideredBot"

// Crawler ..
type Crawler struct {
	Domain          *url.URL
	CustomUserAgent string
	IgnoreRobots    bool
	RobotsData      *robotstxt.RobotsData
	SocketLimit     int
	Entities        map[string]models.PageEntity
	Mutex           *sync.Mutex
	CrawlTime       time.Duration
}

// NewCrawler creates a new Crawler with the specified arguments
func NewCrawler(domain *url.URL, socketLimit int, ignoreRobots bool, userAgent string) (crawler *Crawler) {
	crawler = &Crawler{
		Domain:          domain,
		CustomUserAgent: customUserAgent,
		IgnoreRobots:    ignoreRobots,
		SocketLimit:     socketLimit,
		Entities:        make(map[string]models.PageEntity),
		Mutex:           &sync.Mutex{},
	}

	// Set custom useragent
	if userAgent != "" {
		crawler.CustomUserAgent = userAgent
	}

	log.Infoln(fmt.Sprintf("Created a new Crawler - url: %s, socketLimit: %d, ignoreRobots: %t, customUserAgent: %s", domain.String(), socketLimit, ignoreRobots, crawler.CustomUserAgent))
	return crawler
}

// Crawl starts crawling the specified domain.
func (crawler Crawler) Crawl() {
	var wg sync.WaitGroup
	completedURLs := set.New()
	socketLimit := make(chan int, crawler.SocketLimit)
	for i := 0; i < crawler.SocketLimit; i++ {
		socketLimit <- 1
	}

	if !crawler.IgnoreRobots {
		resp, err := MakeRequest("GET", crawler.Domain.String()+"robots.txt", crawler.CustomUserAgent)
		if err != nil {
			log.Fatalln(err)
			return
		}

		defer resp.Body.Close()
		robots, err := robotstxt.FromResponse(resp)
		if err != nil {
			log.Fatalln(err)
			return
		}

		crawler.RobotsData = robots
	}

	// Start Stopwatch
	startTime := time.Now().UTC()
	wg.Add(1)
	go crawlURL(&crawler, crawler.Domain, completedURLs, &wg, socketLimit)
	wg.Wait()
	crawler.CrawlTime = time.Now().UTC().Sub(startTime)
}

func crawlURL(crawler *Crawler, url *url.URL, completedURLs *set.Set, wg *sync.WaitGroup, socketLimit chan int) {
	defer wg.Done()
	urlStr := url.String()
	completedURLs.Add(urlStr)
	log.Infoln(fmt.Sprintf("Crawling new URL: %s", urlStr))

	<-socketLimit
	resp, err := MakeRequest("GET", urlStr, crawler.CustomUserAgent)
	socketLimit <- 1
	if err != nil {
		log.Errorln(err)
		completedURLs.Remove(urlStr)
		return
	}
	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	for {
		tokenType := tokenizer.Next()
		switch {
		case tokenType == html.ErrorToken:
			crawler.Mutex.Lock()
			crawler.Entities[urlStr] = models.PageEntity{
				URL:         url,
				Path:        url.Path,
				StatusCode:  resp.StatusCode,
				ContentType: resp.Header.Get("Content-Type"),
				ContentSize: resp.ContentLength,
			}
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

			url := ConvertToURL(href, crawler.Domain)
			if !IsSatisfiedURL(crawler.Domain, url) {
				log.Infoln(fmt.Sprintf("Skipping non-satisfied URL: %s", urlStr))
				continue
			}

			if crawler.IgnoreRobots || !crawler.RobotsData.TestAgent(url.Path, crawler.CustomUserAgent) {
				log.Infoln(fmt.Sprintf("Skipping URL as per robot: %s", urlStr))
				continue
			}

			if completedURLs.Has(url.String()) {
				log.Infoln(fmt.Sprintf("URL already crawled: %s", urlStr))
				continue
			}

			wg.Add(1)
			go crawlURL(crawler, url, completedURLs, wg, socketLimit)
			break
		}
	}
}