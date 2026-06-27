package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadPhotos saves listing photos into dir and returns local file paths.
func (c *Client) DownloadPhotos(ctx context.Context, listing *Listing, dir string) ([]string, error) {
	if len(listing.Photos) == 0 {
		return nil, fmt.Errorf("no photos to download")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	var saved []string
	for i, photoURL := range listing.Photos {
		path, err := c.downloadPhoto(ctx, photoURL, dir, i+1)
		if err != nil {
			return saved, fmt.Errorf("photo %d: %w", i+1, err)
		}
		saved = append(saved, path)
	}
	return saved, nil
}

func (c *Client) downloadPhoto(ctx context.Context, photoURL, dir string, index int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, photoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", baseURL+"/")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	ext := photoExt(photoURL, resp.Header.Get("Content-Type"))
	name := fmt.Sprintf("%03d%s", index, ext)
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, io.LimitReader(resp.Body, 20<<20)); err != nil {
		return "", err
	}
	return path, nil
}

func photoExt(urlStr, contentType string) string {
	lower := strings.ToLower(urlStr)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return ".png"
	case strings.HasSuffix(lower, ".webp"):
		return ".webp"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	default:
		return ".jpg"
	}
}
