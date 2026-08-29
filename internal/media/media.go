// Package media is the shared response contract for every platform scraper.
//
// Internally a scraper fills a flat Items slice (each tagged with a Type). The
// public JSON is derived from Items by MarshalJSON according to the Media's
// shape, so existing API consumers keep the exact response they had before the
// scrapers were unified onto this type.
package media

import "encoding/json"

type Type string

const (
	Video Type = "video"
	Audio Type = "audio"
	Image Type = "image"
	Slide Type = "slide"
)

// Item is one downloadable media unit. Superset of the old types.DownloadItem
// (Label/URL/Type/Quality) plus optional Thumbnail/Width/Height for richer
// sources; the extra fields are omitempty so they never appear for scrapers
// that don't set them.
type Item struct {
	Label     string `json:"label,omitempty"`
	URL       string `json:"url"`
	Type      Type   `json:"type"` // no omitempty: matches legacy DownloadItem byte-for-byte
	Quality   string `json:"quality,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
}

// shape selects which legacy JSON view MarshalJSON emits. It exists only to keep
// the pre-refactor response byte-compatible per platform family.
//
// ponytail: one shape per public contract that already shipped. A future v2 API
// can emit the flat Items directly and retire these — not now (no v2 consumer).
type shape int

const (
	shapeDownloads shape = iota // {platform,title,thumbnail,downloads:[Item]}
)

type Media struct {
	Platform  string
	Title     string
	Thumbnail string
	Items     []Item

	shape shape
}

// Downloads builds a Media that marshals to the legacy
// {platform,title,thumbnail,downloads} shape used by reddit, pinterest,
// spotify, soundcloud, terabox, etc. Pass the items the scraper found.
func Downloads(platform, title, thumbnail string, items []Item) *Media {
	return &Media{
		Platform:  platform,
		Title:     title,
		Thumbnail: thumbnail,
		Items:     items,
		shape:     shapeDownloads,
	}
}

func (m *Media) MarshalJSON() ([]byte, error) {
	switch m.shape {
	default: // shapeDownloads
		return json.Marshal(downloadsView{
			Platform:  m.Platform,
			Title:     m.Title,
			Thumbnail: m.Thumbnail,
			Downloads: m.Items,
		})
	}
}

// downloadsView mirrors the old per-platform `Result` struct exactly (field
// order + json tags) so the serialized bytes are unchanged. Downloads is a plain
// slice: a nil Items still marshals to `null`, matching the old code which
// appended onto a nil slice.
type downloadsView struct {
	Platform  string `json:"platform"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Downloads []Item `json:"downloads"`
}
