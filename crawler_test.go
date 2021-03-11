package crawler

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getBaseURLStr() string {
	baseUrl, ok := os.LookupEnv("CRAWLER_BASE_URL")
	if !ok {
		baseUrl = "http://localhost:8080/"
	}
	return baseUrl
}

func getURL(urlStr string) *url.URL {
	baseURL, _ := url.Parse(urlStr)
	return baseURL
}

func getBaseURLIndex() string {
	return getBaseURLStr() + "index.html"
}

// func TestRecursiveCrawler(t *testing.T) {

// 	crawler = NewCrawler()
// 	// fetch the data recursively
// 	result := crawler.Crawl(getBaseUrl())

// 	assert.Equal(t, result.Pages, 5, "Number of pages does not match")
// }

func TestHTMLLinksParsingPagesInSameDirectory(t *testing.T) {

	html := `<!doctype html>
			<html>
			<head></head>
			<body>
				<p>index</p>
				<p><a href="index.html">index</a></p>
				<p><a href="page1.html">p1</a></p>
				<p><a href="page2.html">p2</a></p>
				<p><a href="page3.html">p3</a></p>
			</body>
			</html>`

	baseURL, err := url.Parse("http://localhost:8080/index.html")
	assert.Equal(t, err, nil)
	links := parseLinksFromHTML(baseURL, html)
	assert.Equal(t, len(links), 4)
	assert.Contains(t, links, "http://localhost:8080/index.html")
	assert.Contains(t, links, "http://localhost:8080/page1.html")
	assert.Contains(t, links, "http://localhost:8080/page2.html")
	assert.Contains(t, links, "http://localhost:8080/page3.html")
}

func TestHTMLLinksWithRelativeLinks(t *testing.T) {

	html := `<!doctype html>
			<html>
			<head></head>
			<body>
				<p>index</p>
				<p><a href="child/page1.html">p1</a></p>
				<p><a href="page2.html">p2</a></p>
				<p><a href="page3.html">p3</a></p>
			</body>
			</html>`

	baseURL, err := url.Parse("http://localhost:8080/index.html")
	assert.Equal(t, err, nil)
	links := parseLinksFromHTML(baseURL, html)
	assert.Equal(t, len(links), 3)
	assert.Contains(t, links, "http://localhost:8080/child/page1.html")
	assert.Contains(t, links, "http://localhost:8080/page2.html")
	assert.Contains(t, links, "http://localhost:8080/page3.html")
}

func TestHTMLLinksWithLocalAbsoluteLinks(t *testing.T) {

	html := `<!doctype html>
			<html>
			<head></head>
			<body>
				<p>p11</p>
				<p><a href="/index.html">index</a></p>
				<p><a href="/page1.html">p1</a></p>
				<p><a href="/page2.html">p2</a></p>
				<p><a href="/page3.html">p3</a></p>
			</body>
			</html>`

	baseURL, err := url.Parse("http://localhost:8080/child/page11.html")
	assert.Equal(t, err, nil)
	links := parseLinksFromHTML(baseURL, html)
	assert.Equal(t, len(links), 4)
	assert.Contains(t, links, "http://localhost:8080/index.html")
	assert.Contains(t, links, "http://localhost:8080/page1.html")
	assert.Contains(t, links, "http://localhost:8080/page2.html")
	assert.Contains(t, links, "http://localhost:8080/page3.html")
}

func TestHTMLLinksWithAbsoluteLinks(t *testing.T) {

	html := `<!doctype html>
			<html>
			<head></head>
			<body>
				<p>p11</p>
				<p><a href="http://test:8032/parent/test.html">index</a></p>
				<p><a href="/page1.html">p1</a></p>
				<p><a href="/page2.html">p2</a></p>
				<p><a href="/page3.html">p3</a></p>
			</body>
			</html>`

	baseURL, err := url.Parse("http://localhost:8080/child/page11.html")
	assert.Equal(t, err, nil)
	links := parseLinksFromHTML(baseURL, html)
	assert.Equal(t, len(links), 4)
	assert.Contains(t, links, "http://test:8032/parent/test.html")
	assert.Contains(t, links, "http://localhost:8080/page1.html")
	assert.Contains(t, links, "http://localhost:8080/page2.html")
	assert.Contains(t, links, "http://localhost:8080/page3.html")
}

func TestIntegrationDownloadAndLinks(t *testing.T) {
	dir := "./tmp/"
	_ = os.Mkdir(dir, 0777)
	defer os.RemoveAll(dir) // clean up
	links, err := downloadHTMLAndRetrieveLinks(getBaseURLIndex(), dir)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(links), 4)
	assert.Contains(t, links, getBaseURLIndex())
	assert.Contains(t, links, getBaseURLStr() + "page1.html")
	assert.Contains(t, links, getBaseURLStr() + "page2.html")
	assert.Contains(t, links, getBaseURLStr() + "page3.html")
	fileInfo, err := os.Stat(dir + "index.html")
	assert.Equal(t, err, nil)
	assert.Equal(t, fileInfo.Name(), "index.html")

}

func TestValidURLs(t *testing.T) {

	tests := []struct {
		rootURLStr string
		linkURLStr string
		want       bool
		name       string
	}{
		{"http://localhost:8080/index.html", "http://localhost:8080/page1.html", true, "nominalValid1"},
		{"http://localhost:8080/index.html", "http://localhost:8080/test/page1.html", true, "nominalValid2"},
		{"http://mywebsite/index.html", "http://mywebsite/a/sophisticated/path/page1.html", true, "nominalValid3"},
		{"http://localhost:8080/test/page1.html", "http://localhost:8080/index.html", false, "InvalidDueToNotSubPath"},
		{"https://localhost:8080/index.html", "http://localhost:8080/test/page1.html", false, "InvalidDueToSchemaMismatch"},
		{"http://localhost:8080/index.html", "http://api:8080/test/page1.html", false, "InvalidDueToHostMismatch1"},
		{"http://localhost:8080/index.html", "http://localhost:8081/test/page1.html", false, "InvalidDueToHostMismatch2"},
		{"http:/localhost:8080/index.html", "http://localhost:8080/test/page1.html", false, "InvalidDueToInvalidRootURL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, validLink(test.rootURLStr, test.linkURLStr), test.want)
		})
	}

}

func TestFilterValidURLs(t *testing.T) {

	tests := []struct {
		rootURLStr  string
		linkURLStrs []string
		validURLs   []string
		name        string
	}{
		{"http://localhost:8080/index.html", []string{"http://localhost:8080/page1.html", "http://localhost:8080/page2.html", "http://webservice/page1.html"}, []string{"http://localhost:8080/page1.html", "http://localhost:8080/page2.html"}, "nominalValid1"},
		{"http://localhost:8080/index.html", []string{"http://webservice/page1.html"}, []string{}, "emptyValidURL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, filterValidURLs(test.rootURLStr, test.linkURLStrs), test.validURLs)
		})
	}

}

func TestIntegrationCrawler(t *testing.T) {
	dir := "./tmp/"
	_ = os.Mkdir(dir, 0777)
	defer os.RemoveAll(dir) // clean up
	ctx := context.Background()
	crawler := NewCrawler(ctx, dir, 1000)
	err := crawler.Crawl(getURL(getBaseURLIndex()))
	assert.Equal(t, err, nil)

	expectedPathAndName := []struct {
		path string
		name string
	} {
		{dir+"index.html", "index.html"},
		{dir+"page1.html", "page1.html"},
		{dir+"page2.html", "page2.html"},
		{dir+"page3.html", "page3.html"},
		{dir+"child/page11.html", "page11.html"},
	}

	for _, pathAndName := range expectedPathAndName {
		fileInfo, err := os.Stat(pathAndName.path)
		assert.Equal(t, err, nil)
		assert.Equal(t, fileInfo.Name(), pathAndName.name)
	}
}
