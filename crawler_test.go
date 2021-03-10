package crawler

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getBaseUrl() string {
	baseUrl, ok := os.LookupEnv("CRAWLER_BASE_URL")
	if !ok {
		baseUrl = "http://localhost:8080/index.html"
	}
	return baseUrl
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
	links, err := downloadHTMLAndRetrieveLinks("http://localhost:8080/index.html", dir)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(links), 4)
	assert.Contains(t, links, "http://localhost:8080/index.html")
	assert.Contains(t, links, "http://localhost:8080/page1.html")
	assert.Contains(t, links, "http://localhost:8080/page2.html")
	assert.Contains(t, links, "http://localhost:8080/page3.html")
	fileInfo, err := os.Stat(dir + "index.html")
	assert.Equal(t, err, nil)
	assert.Equal(t, fileInfo.Name(), "index.html")

}

func TestValidURLs(t *testing.T) {

	tests := []struct {
		rootURLStr string
		linkURLStr string
		want        bool
		name string
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
