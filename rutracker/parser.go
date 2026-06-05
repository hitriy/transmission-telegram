package rutracker

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var (
	wsRegex    = regexp.MustCompile(`\s+`)
	scriptRe   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
)

func ParseSearch(rawHTML string) []Torrent {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return nil
	}

	var results []Torrent
	doc.Find("#tor-tbl tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 10 {
			return
		}

		titleCell := cells.Eq(3)
		link := titleCell.Find("a[data-topic_id]").First()
		id, ok := link.Attr("data-topic_id")
		if !ok || id == "" {
			return
		}

		size, _ := strconv.ParseInt(cells.Eq(5).AttrOr("data-ts_text", "0"), 10, 64)
		seeds, _ := strconv.Atoi(strings.TrimSpace(cells.Eq(6).Find("b").First().Text()))
		leeches, _ := strconv.Atoi(strings.TrimSpace(cells.Eq(7).Text()))
		downloads, _ := strconv.Atoi(strings.TrimSpace(cells.Eq(8).Text()))
		registeredUnix, _ := strconv.ParseInt(cells.Eq(9).AttrOr("data-ts_text", "0"), 10, 64)

		results = append(results, Torrent{
			ID:         id,
			Title:      strings.TrimSpace(link.Text()),
			Category:   strings.TrimSpace(cells.Eq(2).Find(".f-name a").First().Text()),
			Author:     strings.TrimSpace(cells.Eq(4).Find("a").First().Text()),
			Size:       size,
			Seeds:      seeds,
			Leeches:    leeches,
			Downloads:  downloads,
			State:      cells.Eq(1).AttrOr("title", ""),
			Registered: time.Unix(registeredUnix, 0),
		})
	})

	return results
}

func ParseMagnetLink(rawHTML string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return ""
	}

	href, ok := doc.Find("a.magnet-link").First().Attr("href")
	if !ok {
		return ""
	}
	return href
}

func ParseDescription(rawHTML string, maxRunes int) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		return ""
	}

	body := doc.Find("div.post_body").First()
	if body.Length() == 0 {
		return ""
	}

	htmlContent, err := body.Html()
	if err != nil {
		return truncateRunes(strings.TrimSpace(body.Text()), maxRunes)
	}

	text := htmlToText(htmlContent)
	return truncateRunes(text, maxRunes)
}

func htmlToText(src string) string {
	src = scriptRe.ReplaceAllString(src, "")
	src = styleRe.ReplaceAllString(src, "")
	src = strings.ReplaceAll(src, "<br>", "\n")
	src = strings.ReplaceAll(src, "<br/>", "\n")
	src = strings.ReplaceAll(src, "<br />", "\n")
	src = strings.ReplaceAll(src, "<hr>", "\n")
	src = strings.ReplaceAll(src, "<hr/>", "\n")
	src = strings.ReplaceAll(src, "<hr />", "\n")

	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return wsRegex.ReplaceAllString(src, " ")
	}

	var buf bytes.Buffer
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "li" || n.Data == "tr") {
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	text := wsRegex.ReplaceAllString(buf.String(), " ")
	text = strings.TrimSpace(text)
	return text
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}
