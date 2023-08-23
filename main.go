package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/html"
)

var globalLinks map[string]struct{}
var globalHost *url.URL
var globalDepth int

type Loc struct {
	Url string `xml:"loc"`
}

type XmlMap struct {
	XMLName xml.Name `xml:"urlset"`
	XlmnsName string `xml:"xlmns,attr"`
	Urls []Loc `xml:"url"`
}

func main() {
	address := flag.String("site", "", "Set the site address")
	depthCount := flag.Int("depth", 0, "Set depth of searching")
	flag.Parse()
	if *address == "" {
		fmt.Println("Pls set the site address")
		return
	}
	globalDepth = *depthCount
	globalLinks = make(map[string]struct{})
	var err error
	globalHost, err = url.Parse(*address)
	if err != nil {
		panic(err)
	}
	HtmlExplorer(*address, 1)
	xmlMap := XmlMap{
		XlmnsName: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}
	for key := range globalLinks {
		xmlMap.Urls = append(xmlMap.Urls, Loc{Url: key})
	}
	output, err := xml.MarshalIndent(xmlMap, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	os.Stdout.Write(output)
}



func HtmlExplorer(address string, depth int) {
	if globalDepth != 0 && depth > globalDepth {
		return
	}
	response, err := http.Get(address)
	if err != nil {
		fmt.Println("Something went wrong while requesting address: ", address)
		fmt.Println(err)
		return
	}
	if response.StatusCode != 200 {
		fmt.Printf("\nAddress: %s status code not OK: %d", address, response.StatusCode)
		return
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	response.Body.Close()

	links := make(map[string]struct{})

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		panic(err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					u, err := url.Parse(attr.Val)
					if err != nil {
						panic(err)
					}
					if (u.Scheme == "http" || u.Scheme == "https") && u.Host == globalHost.Host {
						url := u.String()
						if url[len(url)-1] == '/' {
							url = url[:len(url)-1]
						}
						if _, ok := globalLinks[url]; !ok {
							links[url] = struct{}{}
						}
					} else if u.Scheme == "" && u.Host == "" && u.Path != "/" {
						tempURL := *globalHost
						tempURL.Path = u.Path
						url := tempURL.String()
						if url[len(url)-1] == '/' {
							url = url[:len(url)-1]
						}
						if _, ok := globalLinks[url]; !ok {
							links[url] = struct{}{}
						}
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	for key := range links {
		globalLinks[key] = struct{}{}
	}

	nextDepth := depth + 1

	for key := range links {
		HtmlExplorer(key, nextDepth)
	}
}