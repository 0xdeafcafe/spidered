package main

import (
	"fmt"
	"net/url"
	"os"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"

	"github.com/0xdeafcafe/spidered/crawler"
)

func main() {
	var logLevel string
	var strURL string
	var socketLimit int
	var ignoreRobots bool
	var customUserAgent string

	app := cli.NewApp()
	app.Name = "spidered"
	app.Version = "1.0.0"
	app.Usage = "crawl a domain of your choice"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "log-level, l",
			Usage:       "The level to set the logger (Debug, Info, Warning, Error, Fatal, or Panic)",
			Value:       "Error",
			Destination: &logLevel,
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:  "crawl",
			Usage: "Crawl a url to find every url on the domain.",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "url, u",
					Usage:       "The url to crawl - eg. tomblomfield.com",
					Destination: &strURL,
				},
				cli.IntFlag{
					Name:        "socket-limit, s",
					Value:       15,
					Usage:       "The max number of socket connections to allow.",
					Destination: &socketLimit,
				},
				cli.BoolFlag{
					Name:        "ignore-robots, r",
					Usage:       "If the crawler should ignore a domains robots.txt file.",
					Destination: &ignoreRobots,
				},
				cli.StringFlag{
					Name:        "custom-useragent, ua",
					Usage:       "The UserAgent the bot should send when crawling.",
					Value:       "Googlebot",
					Destination: &customUserAgent,
				},
			},
			Action: func(c *cli.Context) error {
				if !validateAndSetLogLevel(logLevel) {
					fmt.Println("Invalid log level. Reference help.")
					return nil
				}

				if strURL == "" {
					log.Errorln("You must provide a url. Reference help.")
					return nil
				}

				if socketLimit <= 0 {
					log.Errorln("You must provide a socket limit greater than 0.")
					return nil
				}

				url, err := url.Parse(strURL)
				if err != nil {
					log.Errorln("The given url is not valid.")
					return nil
				}

				crawl := crawler.NewCrawler(url, socketLimit, ignoreRobots, "")
				crawl.Crawl()

				for _, entity := range crawl.Entities {
					fmt.Println()
					fmt.Println(fmt.Sprintf("URL: %s", entity.URL))
					fmt.Println(fmt.Sprintf("   - Path: %s", entity.Path))
					fmt.Println(fmt.Sprintf("   - Crawled At: %s", entity.CrawledAt))
					fmt.Println(fmt.Sprintf("   - Content-Type: %s", entity.ContentType))
					fmt.Println(fmt.Sprintf("   - Response Status: %d", entity.ResponseStatus))
					fmt.Println(fmt.Sprintf("   - Response Size (bytes): %d", entity.ResponseSize))
					fmt.Println(fmt.Sprintf("   - Response Checksum (CRC32-IEEE): %d", entity.ResponseChecksum))
					fmt.Println(fmt.Sprintf("   - Response Headers: (%d)", len(entity.ResponseHeaders)))
					for header, headerValues := range entity.ResponseHeaders {
						fmt.Println(fmt.Sprintf("       - %s: %s", header, headerValues))
					}
				}

				return nil
			},
		},
	}

	app.Run(os.Args)
}
