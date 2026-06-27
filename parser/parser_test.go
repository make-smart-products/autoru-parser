package parser

import (
	"strings"
	"testing"
)

const sampleHTML = `
<html><head>
<meta name="description" content="Mazda 3 2012 за 745 000 рублей, 1133169996 в Сергиевом Посаде на Авто.ру"/>
<link href="/catalog/cars/mazda/3/7754738/7754744/specifications/7754744_8510535_7754757/"/>
</head><body>
<h1>Mazda 3 II (BL) Рестайлинг, 2012</h1>
<div data-testid="cardInfoSummary">
  <h3>Владение</h3>
  <ul><li><div class="label">Пробег</div><div class="content">170 000 км</div></li></ul>
  <h3>Характеристики</h3>
  <ul><li><div class="cellTitle">Цвет</div><span class="cellValue">чёрный</span></li></ul>
</div>
<img src="//avatars.mds.yandex.net/get-autoru-vos/19793268/8a85b19f6170d94f57be056409828ac7/584x438"/>
</body></html>
`

func TestParseHTML(t *testing.T) {
	listing, err := parseHTML("https://auto.ru/cars/used/sale/mazda/3/1133169996-dbffd21a/", sampleHTML)
	if err != nil {
		t.Fatal(err)
	}
	if listing.Title != "Mazda 3 II (BL) Рестайлинг, 2012" {
		t.Fatalf("title: %q", listing.Title)
	}
	if listing.OfferID != "1133169996" {
		t.Fatalf("offer id: %q", listing.OfferID)
	}
	if listing.Price != 745000 {
		t.Fatalf("price: %d", listing.Price)
	}
	if listing.Ownership["Пробег"] != "170 000 км" {
		t.Fatalf("mileage: %q", listing.Ownership["Пробег"])
	}
	if listing.Characteristics["Цвет"] != "чёрный" {
		t.Fatalf("color: %q", listing.Characteristics["Цвет"])
	}
	if len(listing.Photos) != 1 {
		t.Fatalf("photos: %d", len(listing.Photos))
	}
	if listing.CatalogIDs == nil || listing.CatalogIDs.ComplectationID != 8510535 {
		t.Fatalf("catalog ids: %+v", listing.CatalogIDs)
	}
}

func TestExtractCatalogIDs(t *testing.T) {
	html := `<a href="/catalog/cars/mazda/3/7754738/7754744/specifications/7754744_8510535_7754757/">`
	ids := extractCatalogIDs(html)
	if ids == nil || ids.TechParamID != 7754757 {
		t.Fatalf("unexpected ids: %+v", ids)
	}
}

func TestIsBlockedPage(t *testing.T) {
	if !isBlockedPage("<html>SmartCaptcha</html>") {
		t.Fatal("expected captcha detection")
	}
	if isBlockedPage(sampleHTML) {
		t.Fatal("sample should not be blocked")
	}
}

func TestAllSpecs(t *testing.T) {
	listing := &Listing{
		Ownership:       map[string]string{"Пробег": "170 000 км"},
		Characteristics: map[string]string{"Цвет": "чёрный"},
		TechSpecs: []SpecGroup{{
			Name: "Двигатель",
			Fields: []SpecItem{{Name: "Объем", Value: "1.6 л"}},
		}},
	}
	specs := listing.AllSpecs()
	if specs["Владение: Пробег"] != "170 000 км" {
		t.Fatalf("ownership spec missing: %+v", specs)
	}
	if specs["Двигатель: Объем"] != "1.6 л" {
		t.Fatalf("tech spec missing: %+v", specs)
	}
}

func TestParseTechInfo(t *testing.T) {
	raw := `{"data":{"tech_info_group":[{"id":"ENGINE","name":"Двигатель","entity":[{"id":"displacement","name":"Объем","value":"1.6","units":"л"}]}]}}`
	groups, err := parseTechInfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || !strings.Contains(groups[0].Fields[0].Value, "л") {
		t.Fatalf("unexpected groups: %+v", groups)
	}
}
