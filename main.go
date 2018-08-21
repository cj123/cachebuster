package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/mattn/go-zglob"
	"golang.org/x/net/html"
)

var (
	filesGlob      string
	version        string
	queryStringKey string
	ignoreRegexStr string
	ignoreRegex    *regexp.Regexp
)

func init() {
	flag.StringVar(&filesGlob, "files", "**/*.html", "glob of files to cache bust")
	flag.StringVar(&version, "version", time.Now().Format("2006010215040507"), "version to update")
	flag.StringVar(&queryStringKey, "queryKey", "cb", "query string key for asset versioning")
	flag.StringVar(&ignoreRegexStr, "ignore", "", "urls to ignore")

	flag.Parse()

	ignoreRegex = regexp.MustCompile(ignoreRegexStr)
}

func main() {
	files, err := zglob.Glob(filesGlob)
	checkError(err, "find files to cachebust")

	for _, file := range files {
		err := bustFile(file)
		checkError(err, "bust file: "+file)
	}
}

func bustFile(filename string) error {
	f, err := os.OpenFile(filename, os.O_RDWR, 0)

	if err != nil {
		return err
	}

	defer f.Close()

	parsed, err := html.Parse(f)

	if err != nil {
		return err
	}

	recurseTree(parsed)

	if err := f.Sync(); err != nil {
		return err
	}

	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	return html.Render(f, parsed)
}

func addVersionToURL(val *url.URL) string {
	q := val.Query()
	q.Set(queryStringKey, version)
	val.RawQuery = q.Encode()

	return val.String()
}

func processAttr(attr *html.Attribute) {
	val, err := url.Parse(attr.Val)

	if err != nil {
		// skip this resource
		return
	}

	if val.Host != "" {
		// only version local files
		return
	}

	if ignoreRegex.MatchString(val.String()) {
		return
	}

	attr.Val = addVersionToURL(val)
}

func recurseTree(n *html.Node) {
	if n.Data == "script" || n.Data == "img" {
		for attrIndex, attr := range n.Attr {
			if attr.Key == "src" {
				processAttr(&n.Attr[attrIndex])
			}
		}
	}

	if n.Data == "link" {
		for attrIndex, attr := range n.Attr {
			if attr.Key == "href" {
				processAttr(&n.Attr[attrIndex])
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		recurseTree(c)
	}
}

func checkError(err error, what string) {
	if err == nil {
		return
	}

	log.Fatalf("could not: %s, err: %s", what, err.Error())
}
