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
	shapeMeta                   // {platform,caption,owner,duration,audio,photos,videos,raw:[MediaItem]}
)

type Media struct {
	Platform  string
	Title     string
	Caption   string
	Thumbnail string
	Items     []Item

	// meta-family scalars (Instagram/Facebook/Threads); ignored by other shapes.
	Owner    string
	Duration string
	Audio    string

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

// Meta builds a Media for the Instagram/Facebook/Threads family, marshaling to
// the legacy {platform,caption,owner,duration,audio,photos,videos,raw} shape.
// photos/videos are derived from each item's Type; raw is every item in the
// order given (append videos before photos to match the pre-refactor output).
func Meta(platform, caption string, items []Item, opts ...MetaOption) *Media {
	m := &Media{Platform: platform, Caption: caption, Items: items, shape: shapeMeta}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type MetaOption func(*Media)

func WithDuration(d string) MetaOption { return func(m *Media) { m.Duration = d } }
func WithAudio(a string) MetaOption    { return func(m *Media) { m.Audio = a } }
func WithOwner(o string) MetaOption    { return func(m *Media) { m.Owner = o } }

func (m *Media) MarshalJSON() ([]byte, error) {
	switch m.shape {
	case shapeMeta:
		v := metaView{Platform: m.Platform, Caption: m.Caption, Owner: m.Owner, Duration: m.Duration, Audio: m.Audio, Raw: metaItems(m.Items)}
		for _, it := range m.Items {
			switch it.Type {
			case Video:
				v.Videos = append(v.Videos, metaItemOf(it))
			case Image:
				v.Photos = append(v.Photos, metaItemOf(it))
			}
		}
		return json.Marshal(v)
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

// metaView mirrors the old FacebookResult/InstagramResult envelope. One struct
// covers both families: field order is caption,owner,duration,audio, and each
// family leaves the other's scalar empty (omitempty), so Facebook emits
// ...caption,duration,audio... and Instagram emits ...caption,owner,audio... —
// byte-identical to the two original structs.
type metaView struct {
	Platform string     `json:"platform"`
	Caption  string     `json:"caption,omitempty"`
	Owner    string     `json:"owner,omitempty"`
	Duration string     `json:"duration,omitempty"`
	Audio    string     `json:"audio,omitempty"`
	Photos   []metaItem `json:"photos,omitempty"`
	Videos   []metaItem `json:"videos,omitempty"`
	Raw      []metaItem `json:"raw,omitempty"`
}

// metaItem is the {quality,thumbnail?,url} shape the meta family has always
// emitted (quality has no omitempty; thumbnail is a nullable pointer).
type metaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}

func metaItemOf(it Item) metaItem {
	var thumb *string
	if it.Thumbnail != "" {
		thumb = &it.Thumbnail
	}
	return metaItem{Quality: it.Quality, Thumbnail: thumb, URL: it.URL}
}

func metaItems(items []Item) []metaItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]metaItem, len(items))
	for i, it := range items {
		out[i] = metaItemOf(it)
	}
	return out
}
