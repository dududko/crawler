package crawler

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
)

// // Result of Crawler
type CrawlerResult struct {
	Pages uint32
}

// type crawler struct {
// 	visited map[string]bool
// 	raw chan string
// 	toCrawl chan string
// 	workerPoolSize uint32
// 	ctx context.Context
// }

// func NewCrawler() Crawler {
	
// }

// func (c *crawler) Cancel() { }


// parseLinksFromHTML retrieves all the links in the html string that satisfies:
// 1. is an absolute path link and child of url 
// 2. is a relative path, in such a case the url is prepended to the path
func parseLinksFromHTML(pageURL *url.URL, html string) []string {
	// regex for finding links
	findLinks := regexp.MustCompile("<a.*?href=\"(.*?)\"")

	matches := findLinks.FindAllStringSubmatch(html, -1)

	links := make([]string, 0)

	var err error
	for _, val := range matches {
		var linkURL *url.URL

		if linkURL, err = url.Parse(val[1]); err != nil {
			continue
		}

		if linkURL.IsAbs() {
			links = append(links, linkURL.String())
		} else if (strings.HasPrefix(linkURL.String(), "/")) {
			links = append(links, pageURL.Scheme + "://"+ pageURL.Host + linkURL.String())
		} else {
			basePath := path.Dir(pageURL.Path)
			links = append(links, pageURL.Scheme + "://"+ pageURL.Host + basePath + linkURL.String())
		}
	}

	return links
}

// validLink returns true if
// 1. both URLs are valid parsable URLs
// 2. both URLs share the same scheme
// 3. both URLs share the same Host 
// 4. the root URL directory is a parent of the link URL
func validLink(rootURLStr string, linkURLStr string) bool {
	rootURL, err := url.Parse(rootURLStr)
	if err != nil {
		log.Printf("Error while parsing rootURLStr [%v], error: [%v]", rootURLStr, err)
		return false
	}

	linkURL, err := url.Parse(linkURLStr)
	if err != nil {
		log.Printf("Error while parsing linkURLStr [%v], error: [%v]", linkURLStr, err)
		return false
	}
	
	if rootURL.Scheme != linkURL.Scheme {
		return false
	}
	
	if rootURL.Host != linkURL.Host {
		return false
	}

	dirRoot := path.Dir(rootURL.Path)
	dirLink := path.Dir(linkURL.Path)

	return strings.HasPrefix(dirLink, dirRoot)
}

// downloadHTMLAndRetrieveLinks Download HTML and retrieve links 
// within the page
func downloadHTMLAndRetrieveLinks(urlStr, dirPath string) ([]string, error) {
	links := make([]string, 0)

	// url from urlStr
	pageURL, err := url.Parse(urlStr)
	if err != nil {
		return links, err
	}

	// generate local path
	filepath := dirPath
	if ! strings.HasSuffix(dirPath, "/") {
		filepath = filepath + "/"
	}
	filepath = filepath + pageURL.Path

	// retrieve bytes from url
	resp, err := http.Get(urlStr)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		return links, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return links, err
	}

	links = append(links, parseLinksFromHTML(pageURL, string(bodyBytes))...)

	// create file (truncates if file already exist)
	f, err := os.Create(filepath)
	if err != nil {
		return links, err
	}
	defer f.Close()
	// copy bytes over to the destination file
	_, err = io.Copy(f, bytes.NewReader(bodyBytes))

	return links, err
}

// downloads the URL and search for references inside 
func workerJob(url string, rawUrls chan<- string) {

}

func startConsumers(ctx context.Context, workerPoolSize uint32, toCrawl <-chan string, rawUrls chan<- string) {

	// semaphore regulates maximum amount of go routines 
	// that can be created 
	sem := make(chan bool, workerPoolSize)

	for {
		select {
		case url := <-toCrawl:
			sem <- true // if more than "workerPoolSize" routines are in use this will block
			go func(url string, rawUrls chan<- string) {
				workerJob(url, rawUrls)
				<-sem
			}(url, rawUrls)
		}
	}
}

 
// func (c *crawler) Crawl(ulr string) (CrawlerResult, error) {

    
//     // Start [workerPoolSize] workers
//     for i := 0; i < c.workerPoolSize; i++ {
//         go workerJob(c.ctx, c.toCrawl, c.raw)
//     }

    
//     wg.Wait()     // wait for all workers to be done
//     fmt.Println("Workers done, shutting down!")
// }

// Crawler crawls recursively a ulr passed in input
// and return a CrawlerResult
type Crawler interface {
	Crawl(ulr string) (CrawlerResult, error)
	// Cancel cancels the crawling
	Cancel() 
}