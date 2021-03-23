package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/rbroggi/crawler"
)

func main() {
	rootURL := flag.String("url", "http://localhost:8080/index.html", "URL to be recursively crawled")
	targetDIR := flag.String("out", "./", "The folder path where the files will be downloaded to")
	maxPoolSize := flag.CommandLine.Uint("pool", 100000, "The max number of go-routines that can be created")

	flag.Parse()
	// Create a new context
	ctx := context.Background()
	// Create a new context, with its cancellation function
	// from the original context
	ctx, cancel := context.WithCancel(ctx)

	// create a channel to communicate OS Signals
	c := make(chan os.Signal, 1)
	
	// configure the os.Interrupt (ctrl+c) to be enqueued to the c channel
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	// in case the ctrl-c event is issued we this go-routine will cancel the context and this will gently 
	// stop the crawling
	go func() {
		select {
		case <-c:
			cancel()
			log.Println("Kill")
			time.Sleep(5 * time.Second)
			log.Println("Kill after 5s")
			os.Exit(1)
		case <-ctx.Done():
		}
	}()
	crawler := crawler.NewCrawler(ctx, *targetDIR, uint32(*maxPoolSize))
	baseURL, err := url.Parse(*rootURL)
	if err != nil {
		log.Printf("Error while parsing URL: [%v]", err)
		os.Exit(1)
	}
	err = crawler.Crawl(baseURL)
	if err != nil {
		log.Printf("Error while crawling: [%v]\n", err)
		os.Exit(2)
	}
}