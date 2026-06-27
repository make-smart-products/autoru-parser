package parser

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var equipmentButtonRE = regexp.MustCompile(`^(Безопасность|Комфорт|Салон|Мультимедиа|Обзор|Внешние элементы|Защита от угона)\s+(\d+)$`)

func extractEquipmentSummary(doc *goquery.Document) []EquipmentGroup {
	seen := map[string]struct{}{}
	var groups []EquipmentGroup

	doc.Find("button").Each(func(_ int, btn *goquery.Selection) {
		text := cleanText(btn.Text())
		m := equipmentButtonRE.FindStringSubmatch(text)
		if len(m) != 3 {
			return
		}
		if _, ok := seen[m[1]]; ok {
			return
		}
		seen[m[1]] = struct{}{}
		groups = append(groups, EquipmentGroup{
			Name:  m[1],
			Items: []string{strings.TrimSpace(m[2] + " опций")},
		})
	})

	return groups
}
