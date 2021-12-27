package epubconv

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	epub "github.com/bmaupin/go-epub"
	"github.com/cixtor/readability"
	"golang.org/x/net/html"
)

func ReadabilityArticleToEpub(input *readability.Article, resultChan chan *epub.Epub, errChan chan error) *epub.Epub {
	doc, err := readabilityArticleToEpub(input)
	if err != nil {
		errChan <- err
		return nil
	}

	resultChan <- doc
	return doc
}

func readabilityArticleToEpub(in *readability.Article) (*epub.Epub, error) {
	doc := epub.NewEpub(in.Title)
	doc.SetDescription(in.Excerpt)
	doc.SetAuthor(in.Byline)

	if err := addImages(doc, in.Node); err != nil {
		// Not a mission critical problem if we can't add the images
		log.Println(fmt.Errorf("%w: %s", ErrAddImage, err))
	}

	var body strings.Builder
	if err := html.Render(&body, in.Node); err != nil {
		return nil, fmt.Errorf("could not render html: %s", err)

	}

	if _, err := doc.AddSection(body.String(), "Content", "", ""); err != nil {
		return nil, fmt.Errorf("could not add section: %s", err)
	}
	return doc, nil
}

var ErrAddImage = errors.New("could not add image to epub document")

func addImages(doc *epub.Epub, n *html.Node) error {
	if n.Type == html.ElementNode && n.Data == "img" {
		for i, a := range n.Attr {
			if a.Key == "src" {
				// get the filename
				u, err := url.Parse(a.Val)
				if err != nil {
					return fmt.Errorf("could not parse url: %v", err)
				}
				f := filepath.Base(u.Path)
				img, err := doc.AddImage(a.Val, f)
				if err != nil {
					log.Println("could not add image:", err)
					continue
				}
				n.Attr[i].Val = img
			}
			// remove the srcset
			if a.Key == "srcset" {
				n.Attr[i] = n.Attr[len(n.Attr)-1]        // Copy last element to index i.
				n.Attr[len(n.Attr)-1] = html.Attribute{} // Erase last element (write zero value).
				n.Attr = n.Attr[:len(n.Attr)-1]          // Truncate slice.
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		err := addImages(doc, c)
		if err != nil {
			return fmt.Errorf("could not replace images: %s", err)
		}
	}
	return nil
}
