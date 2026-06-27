package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var spaceRE = regexp.MustCompile(`\s+`)

func parseHTML(pageURL, html string) (*Listing, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	listing := &Listing{
		URL:             pageURL,
		OfferID:         offerID(pageURL),
		Ownership:       map[string]string{},
		Characteristics: map[string]string{},
		Photos:          extractPhotos(html),
		CatalogIDs:      extractCatalogIDs(html),
	}

	listing.Title = cleanText(doc.Find("h1").First().Text())
	listing.Description = metaContent(doc, "description")
	listing.Location = extractLocation(doc, listing.Description)
	listing.Seller = extractSeller(doc)
	listing.Comment = extractComment(doc)
	listing.Price, listing.PriceFormatted = extractPrice(doc, html)

	for _, section := range extractSummarySections(doc) {
		switch section.Title {
		case "Владение":
			for k, v := range section.Items {
				listing.Ownership[k] = v
			}
		case "Характеристики":
			for k, v := range section.Items {
				listing.Characteristics[k] = v
			}
		default:
			for k, v := range section.Items {
				listing.Characteristics[k] = v
			}
		}
	}
	mergeCharacteristics(doc, listing.Characteristics)
	listing.Equipment = extractEquipmentSummary(doc)

	if listing.Title == "" {
		return nil, fmt.Errorf("listing title not found; page may be blocked or captcha served")
	}
	if len(listing.Photos) == 0 {
		if img := metaContent(doc, "og:image"); img != "" {
			listing.Photos = []string{img}
		}
	}

	return listing, nil
}

type summarySection struct {
	Title string
	Items map[string]string
}

func extractSummarySections(doc *goquery.Document) []summarySection {
	var sections []summarySection

	doc.Find(`[data-testid="cardInfoSummary"]`).Each(func(_ int, block *goquery.Selection) {
		block.Find("h3").Each(func(_ int, heading *goquery.Selection) {
			title := cleanText(heading.Text())
			if title == "" {
				return
			}
			items := map[string]string{}
			heading.NextUntil("h3").Find("li").Each(func(_ int, li *goquery.Selection) {
				label := cleanText(li.Find(`[class*="label"]`).First().Text())
				value := cleanText(li.Find(`[class*="content"]`).First().Text())
				if label == "" {
					label = cleanText(li.Find(`[class*="cellTitle"]`).First().Text())
					value = cleanText(li.Find(`[class*="cellValue"]`).First().Text())
				}
				if label != "" && value != "" {
					items[label] = value
				}
			})
			if len(items) > 0 {
				sections = append(sections, summarySection{Title: title, Items: items})
			}
		})
	})

	return sections
}

func mergeCharacteristics(doc *goquery.Document, characteristics map[string]string) {
	doc.Find(`[data-testid="cardInfoSummary"]`).Each(func(_ int, block *goquery.Selection) {
		title := cleanText(block.Find("h3").First().Text())
		if title != "Характеристики" {
			return
		}
		block.Find(`[class*="ComplexRow"]`).Each(func(_ int, row *goquery.Selection) {
			label := cleanText(row.Find(`[class*="label"]`).First().Text())
			value := cleanText(row.Find(`[class*="cellValue"]`).First().Text())
			if label == "" {
				label = cleanText(row.Find(`[class*="content"]`).First().Text())
			}
			if label != "" && value != "" {
				characteristics[label] = value
			}
		})
	})
}

var (
	descPriceRE = regexp.MustCompile(`(?i)за\s+([\d\s` + "\u00a0" + `]+)\s+руб`)
)

func extractPrice(doc *goquery.Document, html string) (int64, string) {
	for _, sel := range []string{
		`[itemprop="price"]`,
		`meta[property="product:price:amount"]`,
	} {
		if content, ok := doc.Find(sel).Attr("content"); ok {
			if price, formatted := parsePriceValue(content); price > 0 && price < 1_000_000_000 {
				return price, formatted
			}
		}
	}

	for _, text := range []string{
		metaContent(doc, "og:description"),
		metaContent(doc, "description"),
	} {
		if m := descPriceRE.FindStringSubmatch(text); len(m) == 2 {
			if price, formatted := parsePriceValue(m[1]); price > 0 {
				return price, formatted
			}
		}
	}

	if m := priceRE.FindAllStringSubmatch(doc.Text(), -1); len(m) > 0 {
		var best int64
		var bestFormatted string
		for _, match := range m {
			price, formatted := parsePriceValue(match[1])
			if price > best && price < 1_000_000_000 {
				best = price
				bestFormatted = formatted
			}
		}
		if best > 0 {
			return best, bestFormatted
		}
	}
	return 0, ""
}

func parsePriceValue(raw string) (int64, string) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, raw)
	if digits == "" {
		return 0, ""
	}
	price, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, ""
	}
	formatted := fmt.Sprintf("%s ₽", formatThousands(price))
	return price, formatted
}

func formatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	if s != "" {
		parts = append([]string{s}, parts...)
	}
	return strings.Join(parts, " ")
}

func extractLocation(doc *goquery.Document, description string) string {
	if loc := locationFromDescription(description); loc != "" {
		return loc
	}
	var location string
	doc.Find(`button`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		text := cleanText(s.Text())
		if strings.Contains(text, "●") || strings.Contains(strings.ToLower(text), "телефон") {
			return true
		}
		if s.Closest(`[class*="Seller"]`).Length() > 0 && len(text) > 2 && len(text) < 80 {
			location = text
			return false
		}
		return true
	})
	return location
}

func locationFromDescription(description string) string {
	idx := strings.LastIndex(description, " в ")
	if idx < 0 {
		return ""
	}
	rest := description[idx+3:]
	if end := strings.Index(rest, " на Авто.ру"); end > 0 {
		return cleanText(rest[:end])
	}
	return ""
}

func extractSeller(doc *goquery.Document) string {
	var seller string
	doc.Find(`a[class*="Seller"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		seller = cleanText(s.Text())
		return seller == ""
	})
	if seller != "" {
		return seller
	}
	doc.Find(`a`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if strings.Contains(href, "/profile/") || strings.Contains(href, "/dealer/") {
			seller = cleanText(s.Text())
			return seller == ""
		}
		return true
	})
	return seller
}

func extractComment(doc *goquery.Document) string {
	var comment string
	doc.Find("h4").EachWithBreak(func(_ int, h *goquery.Selection) bool {
		if cleanText(h.Text()) == "Комментарий продавца" {
			comment = cleanText(h.Parent().Find("p, div").Not("h4").First().Text())
			if comment == "" {
				comment = cleanText(h.Next().Text())
			}
			return false
		}
		return true
	})
	return comment
}

func metaContent(doc *goquery.Document, name string) string {
	if v, ok := doc.Find(fmt.Sprintf(`meta[name="%s"]`, name)).Attr("content"); ok {
		return cleanText(v)
	}
	if v, ok := doc.Find(fmt.Sprintf(`meta[property="%s"]`, name)).Attr("content"); ok {
		return cleanText(v)
	}
	return ""
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.TrimSpace(spaceRE.ReplaceAllString(s, " "))
}
