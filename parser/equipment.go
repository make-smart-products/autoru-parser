package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func (c *Client) fetchCatalogEquipment(ctx context.Context, listing *Listing) ([]EquipmentGroup, error) {
	if listing.CatalogIDs == nil {
		return nil, fmt.Errorf("missing catalog ids")
	}

	catalogURL := catalogEquipmentURL(listing.CatalogIDs)
	html, err := c.fetch(ctx, catalogURL, nil)
	if err != nil {
		return nil, err
	}

	return parseCatalogEquipment(html, listing.Characteristics["Комплектация"])
}

func catalogEquipmentURL(ids *CatalogIDs) string {
	return fmt.Sprintf("%s/catalog/cars/%s/%s/%d/%d/equipment/%d/",
		baseURL, ids.Mark, ids.Model, ids.GenerationID, ids.ConfigurationID, ids.ComplectationID)
}

func parseCatalogEquipment(html, trimName string) ([]EquipmentGroup, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var groups []EquipmentGroup
	doc.Find("h3").Each(func(_ int, h *goquery.Selection) {
		title := cleanText(h.Text())
		if !isEquipmentHeading(title) {
			return
		}
		group := EquipmentGroup{Name: title}
		h.NextUntil("h3").Find("li").Each(func(_ int, li *goquery.Selection) {
			item := cleanText(li.Text())
			if item != "" {
				group.Items = append(group.Items, item)
			}
		})
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	})

	if len(groups) == 0 {
		return nil, fmt.Errorf("equipment groups not found")
	}
	return groups, nil
}

func isEquipmentHeading(title string) bool {
	switch title {
	case "Безопасность", "Комфорт", "Салон", "Мультимедиа", "Обзор", "Внешние элементы", "Защита от угона":
		return true
	default:
		return false
	}
}
