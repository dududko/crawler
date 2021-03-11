# Web crawler

This program is a command line utility to crawl recursively a url. It features a recursive crawling and parallel crawling. Also, the level of parallelism can be set but a sensible default is provided. 
## Design

To enhance testability the program was implemented as a library with it's relative `_test.go` file and only a very small main function which makes use of the library. 
The **Crawler** will follow a best effort approach: it will attempt to download/scrape all the links that are eligible. If some of the HTTP GET fails the crawler will continuing attempting to crawl the rest of the items and will not stop. A different approach could have been used depending on the requirements: as soon as an error occur stop the whole scrapping process.  
In the solution there will be a go-routine that will dispatch the scrapping download of new urls to consumers that are run in lazily initialized go-routines. Those routines will download and enqueue the found links into a channel which will be processed by a 'orchestrator' go-routine which will post-process the candidate urls and decide whether they match the criteria to be scraped:
1. not already scrapped
2. be a sub-ulr of the initial root-url

## Build

To build the command line:

```bash
$ cd cmd && go build -o crawl
```

## Run

To run the binary:

```bash
$ ./crawl -url=<url_to_be_crawled> -out="<directory_where_the_resources_will_be_downloaded>"
```

## Build, unit-test and run with docker-compose

By running the following command the project will be built, will be unit-tested against a dummy local web-server (run within its own docker container) and the the command-line will be launched against this local web-server:

```bash
$ docker-compose up --build --abort-on-container-exit
```

Also a [github actions workflow](https://github.com/features/actions) is put in place for a small CI of this repository.

## Unit Tests

* The unit-tests with prefix `TestIntegration` are only runnable if the dummy HTTP server identified in the `docker-compose.yaml` file as the **site** service is up.
* Even though I support the idea of unit-testing only the public-interface (exported-methods) of a given package as a way to allow implementation details to vary without impacting behavior from customer/client code in this repo I ended-up testing some non-exported methods as a strategy to test some more localized corner-cases without the need to put in place a server or an integration test for that purpose (e.g. the 'TestValidURLs' unit-test )

## limitations

* only links which point to html files are supported
* this program does not support continuous load as the time available was not enough to include this functionality
* the mechanism to flag the natural end of the recursive scraping is a timeout set at consumer level which is triggered if no new url to be scrapped is dispatched within the configured timeout parameter (10 milliseconds). Currently this parameter is not 
easily configurable but it would be easy to extend the program to accommodate this change.
* ctrl+c will be catch and a cleanup will be triggered. The cleanup will:
   1. stop issuing new urls to be scrapped
   2. finish the ongoing urls downloads
   The cleanup will not cancel the ongoing http get calls. 
* the program has still some bugs which I was not able to troubleshoot during the 4 hours reserved to this exercise. Specially this program works well with the dummy web-server sample delivered along in the docker-compose but it fails to work correctly on real sites like `https://golang.org/`
