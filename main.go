// OPML cleaner takes an opml xml file and checks if the
// entries are returning 200 OK
// It will output a new OPML file with the bad entries removed

package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

type Outline struct {
	XMLName     xml.Name `xml:"outline"`
	Text        string   `xml:"text,attr"`
	Title       string   `xml:"title,attr"`
	Description string   `xml:"description,attr"`
	Type        string   `xml:"type,attr"`
	Version     string   `xml:"version,attr"`
	HtmlURL     string   `xml:"htmlUrl,attr"`
	XmlURL      string   `xml:"xmlUrl,attr"`
}

type Head struct {
	XMLName     xml.Name `xml:"head"`
	Title       string   `xml:"title"`
	DateCreated string   `xml:"dateCreated"`
}

type Body struct {
	XMLName xml.Name  `xml:"body"`
	Outline []Outline `xml:"outline"`
}

type Opml struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head
	Body    Body
}

func parseFeed(url string, r io.Reader) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	feed, err := fp.Parse(r)
	if err != nil {
		return nil, err
	}
	return feed, nil
}

// getFeed fetches the feed, parses it and returns a Feed
func getFeed(url string) (*gofeed.Feed, error) {
	// fetch xml from remote
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// if status is not 200 the feed doesn't exist
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("\"%s\": status %d", url, resp.StatusCode)
	}

	// parse feed to check if it's valid
	feed, err := parseFeed(url, resp.Body)
	if err != nil {
		return nil, err
	}

	return feed, nil
}

// readOpml reads an OPML file and returns a Opml struct
func readOpml(filename string) Opml {
	log.Printf("reading %s", filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}
	opml := Opml{}
	err = xml.Unmarshal(data, &opml)
	if err != nil {
		log.Fatal(err)
	}
	return opml
}

func createOpml(feeds []Outline) Opml {
	newOpml := Opml{
		Version: "2.0",
		Head: Head{
			Title:       "feeds",
			DateCreated: time.Now().Format(time.RFC822),
		},
		Body: Body{
			Outline: feeds,
		},
	}
	return newOpml
}

func main() {
	opml := readOpml("rss-export.opml")
	log.Printf("found %d entries", len(opml.Body.Outline))

	numFeeds := len(opml.Body.Outline)

	successFeeds := []Outline{}
	failedFeeds := []Outline{}
	for i := 0; i < numFeeds; i++ {
		// skip outline elements that are not feeds
		entry := opml.Body.Outline[i]
		log.Printf("[%d/%d] %s", i+1, numFeeds, entry.Title)
		// todo remove from numfeeds
		if entry.XmlURL == "" {
			log.Printf("no xml url %s", entry.Title)
			continue
		}

		// fetch and parse feed
		_, err := getFeed(entry.XmlURL)
		if err != nil {
			log.Printf("%s", err)
			failedFeeds = append(failedFeeds, entry)
			continue
		}

		successFeeds = append(successFeeds, entry)
	}
	log.Printf("success: %d failed: %d", len(successFeeds), len(failedFeeds))

	// generate new feed and write to file
	newOpml := createOpml(successFeeds)
	output, err := xml.MarshalIndent(newOpml, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", xml.Header)
	fmt.Printf("%s", output)
}
