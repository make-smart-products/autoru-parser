package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	baseURL     = "https://auto.ru"
	techInfoURL = baseURL + "/-/ajax/desktop-search/getCatalogTechInfo/"
	equipURL    = baseURL + "/-/ajax/desktop-search/getComplectationInfo/"
	userAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var (
	offerIDFromURL = regexp.MustCompile(`/(\d+)-[a-f0-9]+/?$`)
	catalogSpecRE  = regexp.MustCompile(`/catalog/cars/([^/]+)/([^/]+)/(\d+)/(\d+)/specifications/(\d+)_(\d+)_(\d+)/`)
	photoRE        = regexp.MustCompile(`get-autoru-vos/\d+/[a-f0-9]+`)
	priceRE        = regexp.MustCompile(`(\d[\d\s` + "\u00a0" + `]*)\s*₽`)
)

// Client downloads and parses auto.ru car listings.
type Client struct {
	http *http.Client
}

// NewClient creates a parser client with cookie support and sane timeouts.
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{
			Timeout: 45 * time.Second,
			Jar:     jar,
		},
	}
}

// Parse fetches a listing page and enriches it with catalog tech data when available.
func (c *Client) Parse(ctx context.Context, pageURL string) (*Listing, error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return nil, fmt.Errorf("empty url")
	}

	html, err := c.fetch(ctx, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}

	listing, err := parseHTML(pageURL, html)
	if err != nil {
		return nil, err
	}

	if listing.CatalogIDs != nil {
		c.enrichFromAPI(ctx, pageURL, listing)
	}

	return listing, nil
}

func (c *Client) enrichFromAPI(ctx context.Context, pageURL string, listing *Listing) {
	ids := listing.CatalogIDs
	csrf := c.csrfToken()

	techBody, err := json.Marshal(map[string]int{
		"configuration_id": ids.ConfigurationID,
		"complectation_id": ids.ComplectationID,
		"tech_param_id":    ids.TechParamID,
	})
	if err != nil {
		return
	}

	if techRaw, err := c.fetchAPI(ctx, pageURL, techInfoURL, techBody, csrf); err == nil {
		if specs, err := parseTechInfo(techRaw); err == nil {
			listing.TechSpecs = specs
		}
	}

	equipBody, err := json.Marshal(map[string]int{
		"configuration_id": ids.ConfigurationID,
		"complectation_id": ids.ComplectationID,
	})
	if err != nil {
		return
	}

	if equipRaw, err := c.fetchAPI(ctx, pageURL, equipURL, equipBody, csrf); err == nil {
		if equipment, err := parseEquipment(equipRaw); err == nil {
			listing.Equipment = equipment
		}
	}

	if len(listing.Equipment) == 0 {
		if equipment, err := c.fetchCatalogEquipment(ctx, listing); err == nil {
			listing.Equipment = equipment
		}
	}
}

func (c *Client) csrfToken() string {
	u, _ := url.Parse(baseURL)
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "_csrf_token" {
			return cookie.Value
		}
	}
	return ""
}

func (c *Client) fetchAPI(ctx context.Context, referer, target string, body []byte, csrf string) (string, error) {
	return c.fetch(ctx, target, body, referer, csrf)
}

func (c *Client) fetch(ctx context.Context, target string, body []byte, refererAndCSRF ...string) (string, error) {
	var reader io.Reader
	method := http.MethodGet
	if body != nil {
		reader = bytes.NewReader(body)
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/json,*/*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	referer := baseURL + "/"
	csrf := ""
	if len(refererAndCSRF) > 0 && refererAndCSRF[0] != "" {
		referer = refererAndCSRF[0]
	}
	if len(refererAndCSRF) > 1 {
		csrf = refererAndCSRF[1]
	}
	req.Header.Set("Referer", referer)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		if csrf != "" {
			req.Header.Set("X-Csrf-Token", csrf)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d for %s", resp.StatusCode, target)
	}
	return string(data), nil
}

func offerID(urlStr string) string {
	if m := offerIDFromURL.FindStringSubmatch(strings.TrimSuffix(urlStr, "/")); len(m) == 2 {
		return m[1]
	}
	return ""
}

func extractCatalogIDs(html string) *CatalogIDs {
	m := catalogSpecRE.FindStringSubmatch(html)
	if len(m) != 8 {
		return nil
	}
	var ids CatalogIDs
	ids.Mark = m[1]
	ids.Model = m[2]
	fmt.Sscanf(m[3], "%d", &ids.GenerationID)
	fmt.Sscanf(m[4], "%d", &ids.ConfigurationID)
	fmt.Sscanf(m[6], "%d", &ids.ComplectationID)
	fmt.Sscanf(m[7], "%d", &ids.TechParamID)
	if ids.ConfigurationID == 0 {
		return nil
	}
	return &ids
}

func extractPhotos(html string) []string {
	seen := make(map[string]struct{})
	var photos []string
	for _, match := range photoRE.FindAllString(html, -1) {
		full := "https://avatars.mds.yandex.net/" + match + "/1200x900"
		if _, ok := seen[full]; ok {
			continue
		}
		seen[full] = struct{}{}
		photos = append(photos, full)
	}
	return photos
}
