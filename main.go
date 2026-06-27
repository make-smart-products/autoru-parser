package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/make-smart-products/autoru-parser/parser"
)

func main() {
	url := flag.String("url", "", "auto.ru car listing URL")
	out := flag.String("out", "output", "directory for JSON and photos")
	pretty := flag.Bool("pretty", true, "pretty-print JSON")
	noPhotos := flag.Bool("no-photos", false, "skip photo download")
	timeout := flag.Duration("timeout", 2*time.Minute, "request timeout")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: autoru-parser -url <listing-url> [-out output] [-no-photos] [-timeout 2m]")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := parser.NewClient(parser.WithTimeout(*timeout))

	listing, err := client.Parse(ctx, *url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir error: %v\n", err)
		os.Exit(1)
	}

	var saved []string
	if !*noPhotos {
		photoDir := filepath.Join(*out, "photos")
		saved, err = client.DownloadPhotos(ctx, listing, photoDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "photo download warning: %v\n", err)
		}
	}

	result := struct {
		*parser.Listing
		SavedPhotos []string `json:"saved_photos,omitempty"`
		SpecCount   int      `json:"spec_count"`
	}{
		Listing:     listing,
		SavedPhotos: saved,
		SpecCount:   len(listing.AllSpecs()),
	}

	var data []byte
	if *pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
		os.Exit(1)
	}

	jsonPath := filepath.Join(*out, "listing.json")
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saved %s\n", jsonPath)
	if !*noPhotos {
		fmt.Printf("Downloaded %d photos to %s\n", len(saved), filepath.Join(*out, "photos"))
	}
	fmt.Printf("Title: %s\n", listing.Title)
	if listing.PriceFormatted != "" {
		fmt.Printf("Price: %s\n", listing.PriceFormatted)
	}
	fmt.Printf("Specs: %d fields, %d photos\n", result.SpecCount, len(listing.Photos))
}
