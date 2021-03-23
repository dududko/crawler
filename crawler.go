package crawler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

func (c *crawler) Crawl(rootURL *url.URL) error {

	if rootURL == nil {
		return errors.New("Cannot crawl nil *url.URL")
	}

	// channel used to send to consumers the valid urls to be crawled
	// 1. read by the consumers go-routines
	// 2. written by the orchestrator go-routine
	toCrawl := make(chan string)

	// rawUrls channel containing the rawURLs to be scrapped, not all of them will actually be dispatched 
	// 1. read by the orchestrator go-routine
	// 2. written by the consumers go-routines
	rawURLs := make(chan []string)

	// termConsumers is a channel for signaling termination to consumers 
	termConsumers := make(chan struct{})

	// termOrchestrator is a channel for signaling termination to the orchestrator
	termOrchestrator := make(chan struct{})

	// consumersDone is a channel used by the consumer go-routine to signal a 
	// timeout triggered by the absence of new ulr to be scrapped
	consumersDone := make(chan struct{})
	
	// ongoingJobs is used to track the go-routines both in the consumers and the orchestrator side
	var wgConsumers sync.WaitGroup
	wgConsumers.Add(1)
	go func(maxPoolSize uint32) {
		defer wgConsumers.Done()
		startConsumers(c.targetDir, maxPoolSize, toCrawl, rawURLs, termConsumers, consumersDone) 
	}(c.maxPoolSize)

	var wgOrchestrator sync.WaitGroup
	wgOrchestrator.Add(1)
	go func (rootURL *url.URL)  {
		defer wgOrchestrator.Done()
		startOrchestrator(rootURL.String(), toCrawl, rawURLs, termOrchestrator)
	}(rootURL)

	// launch first url to rawURLs channel
	toCrawl <- rootURL.String()

	// blocks until either the naturalEndOfWork or a context.cancel is issued
	select {
	case <-consumersDone:
		log.Println("Natural termination due to no new workload requested")
	case <-c.ctx.Done():
		log.Println("Job Cancelled")
	}

	// signaling two terminations for consumer routine and for orchestration routine
	termConsumers<-struct{}{}
	log.Println("w8 consumers")
	wgConsumers.Wait()
	termOrchestrator<-struct{}{}
	log.Println("w8 orchestrator")
	wgOrchestrator.Wait()

	close(rawURLs)
	close(toCrawl)

	log.Println("exit crawler")

	return nil
}

// Crawler crawls recursively a ulr passed in input
// and return a CrawlerResult
type Crawler interface {
	// Crawl will crawl recursively all the pages under the provided url
	// url must be a valid non-nil url otherwise an empty CrawlerResult and 
	// an error will be returned.
	// Different errors can also be returned (e.g. due to failure in HTTP GET)
	Crawl(rootURL *url.URL) error
}

type crawler struct {
	ctx context.Context
	targetDir string
	maxPoolSize uint32
}

// NewCrawler returns a Crawler object  
func NewCrawler(ctx context.Context, targetDir string, maxPoolSize uint32) Crawler {
	return &crawler {
		ctx, targetDir, maxPoolSize,
	}
}

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

// filterValidURLs return only the valid linkURLStrs tested against 
// the rootURLStr and using the validLink method
func filterValidURLs(rootURLStr string, linkURLStrs []string) []string {
	validURLs := make([]string, 0)
	for _, link := range linkURLStrs {
		if validLink(rootURLStr, link) {
			validURLs = append(validURLs, link)
		}
	}
	return validURLs
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
	filepath := path.Join(dirPath, pageURL.Path)
	log.Printf("Downloading file [%v] into filepath [%v]", urlStr, filepath)

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

	// ensure parent dir exists
	err = os.MkdirAll(filepath, 0755)
	if err != nil {
		return links, err
	}
	// create file (truncates if file already exist)
	f, err := os.Create(path.Join(filepath, "index.html"))
	if err != nil {
		return links, err
	}
	defer f.Close()
	// copy bytes over to the destination file
	_, err = io.Copy(f, bytes.NewReader(bodyBytes))

	return links, err
}

// downloads the URL and search for references inside 
func workerJob(targetDir string, linkURL string, rawUrls chan<- []string) (res []string) {
	res, err := downloadHTMLAndRetrieveLinks(linkURL, targetDir)
	if err != nil {
		log.Printf("Error while downloading and Retrieving links: %v\n", err)
		return
	}
	return
}

// startOrchestrator will listen to 'rawURLs' channel and validate the set of incoming
// urls against the configured rootURL. The valid URLs will then be 'enqueued' into the
// 'toCrawl' channel for being processed by the workers. 
// rootURL is the first url requested for scrapping which will actually be recursively scrapped
// toCrawl is the channel where this orchestrator go-routine will enqueue urls that are eligible to be enqueued
// rawURLs is the channel where the orchestrator go-routine dequeue candidate urls to be enqueued 
// term channel is used to shut down the orchestrator go-routine
func startOrchestrator(rootURL string, toCrawl chan<- string, rawURLs <-chan []string, term <-chan struct{}) {

	crawledURLs := make(map[string]bool)
Loop:
	for {
		select {
		case candidateURLs := <-rawURLs:
			log.Println("read from row urls", candidateURLs)
			validURLs := filterValidURLs(rootURL, candidateURLs)

			for _, validURL := range validURLs {
				_, found := crawledURLs[validURL]
				if ! found {
					crawledURLs[validURL] = true
					log.Printf("Enqueuing url: [%v]", validURL)
					select {
						case toCrawl<- validURL:
						case <-term:
							log.Println("End of orchestrator routine")
							break Loop
					}
				}
			}
			log.Println("continue loop")
		case <-term:
			log.Println("End of orchestrator routine")
			break Loop
		}
	}
}

// startConsumers will listen to the 'toCrawl' channel and spawn a go-routine for each
// new url incoming into this channel. This spawned go-routine will download the incoming
// url and enqueue in the 'rawURLs' the list of ulrs found within the page. those ulrs are 
// only candidates urls and will be processed by the orchestrator go-routine which will 
// arbitrate whether or not this candidate url will be crawled.
// If no incoming task is dispatched to this consumer go-routine within 10 milliseconds it will 
// signal that the consumming job is done by enqueuing into the 'consumerDone' channel
// finally the 'term' channel is used externally to cancell and terminate this go-routine
// targetDir is the target dir where the consumers will dowload the urls into
// workerPoolSize is the amount of maximum concurrent go-routines that can be spawned by the consumer go-routine
// toCrawl is the channel where incoming url to be scraped are dequeued from
// rawURLs is the channel where the candidate urls to be scraped are enqueued
// term is the channel used to terminate the consumer go-routines
// consumerDone is the channel used by the consumer go-routine to signal that the scrapping is done - this is triggered if no job is received in the 'toCrawl' channel within
//              a timeout of 10 milliseconds
func startConsumers(targetDir string, workerPoolSize uint32, toCrawl <-chan string, rawURLs chan<- []string, term <-chan struct{}, consumersDone chan<- struct{}) {

	// sem is used as a semaphore to regulates maximum amount of go routines 
	// that can be created concurrently
	// if workerPoolSize is set to 0 there will be no limit of go-routines
	var sem chan struct{}
	if workerPoolSize == 0 {
		sem = make(chan struct{})
	} else {
		sem = make(chan struct{}, workerPoolSize)
	}
	var wg sync.WaitGroup
Loop:
	for {
		select {
		//case <-time.After(10*time.Millisecond):
		//	consumersDone <- struct{}{}
		case urlToCrawl := <-toCrawl:
			sem <- struct{}{}// if more than "workerPoolSize" routines are in use this will block until one is availlable
			wg.Add(1)
			go func(targetDir string, linkURL string, rawURLs chan<- []string) {
				defer func() {
					log.Println("exit worker")
					wg.Done() // free the go-routine and free the waiting group
					<-sem 	// free slot in pool
					log.Println("exit worker wone")
				}()
				res := workerJob(targetDir, linkURL, rawURLs)
				if len(res) > 0 {
					select {
					case rawURLs <- res:
					case <-term:
						return
					}
				}
			}(targetDir, urlToCrawl, rawURLs)
		case <-term:
			log.Println("End of consumer routine")
			wg.Wait()
			break Loop
		}
	}
}
