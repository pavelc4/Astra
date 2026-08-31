# Astra — Project Context

> Read this before exploring the codebase. Written for agents to avoid wasting tokens.

## Overview

Multi-platform media downloader + scraper REST API written in Go.
Supports: Instagram, TikTok, Twitter, Facebook, Threads, Reddit, Pinterest,
Terabox, Spotify, SoundCloud, CapCut, LinkedIn.

- **Module:** `github.com/pavelc4/astra`
- **Go version:** 1.26.3
- **Server default:** `http://0.0.0.0:3000`
- **Server entry point:** `cmd/server/`
- **Cookie extractor entry point:** `cmd/ig-cookies/`

---

## Directory Structure

```
astra/
├── cmd/
│   ├── server/
│   │   ├── main.go          # HTTP server bootstrap (chi router, slog logger)
│   │   └── routes.go        # All routes registered here
│   └── ig-cookies/
│       ├── main.go          # CLI: -browser, -profile, -exec, -out flags
│       └── extractor/
│           ├── types.go     # InstagramCookies struct + Extractor interface
│           ├── firefox.go   # Backend: read cookies.sqlite directly (no browser launch)
│           └── chromium.go  # Backend: CDP via chromedp (launches browser)
├── internal/
│   ├── config/
│   │   └── config.go        # Loads .env + env vars → instagram.SetCookies()
│   ├── errors/
│   │   └── errors.go        # AppError struct + constructor helpers
│   ├── handler/             # One file per platform, all thin wrappers
│   │   ├── instagram.go
│   │   ├── tiktok.go
│   │   └── ... (11 files total)
│   ├── httpclient/
│   │   └── httpclient.go    # Shared http.Client (15s timeout) + DefaultHeaders
│   ├── instagram/           # Core Instagram package
│   │   ├── api.go           # Public funcs: FetchMedia, FetchProfile, SetCookies
│   │   ├── client.go        # IGClient struct: req(), getJSON(), HasCookies()
│   │   ├── graphql.go       # FetchProfile, FetchMedia, FetchStories (mobile API)
│   │   └── types.go         # MediaItem, UserProfile, MediaInfo, etc.
│   ├── platform/            # One folder per external platform
│   │   ├── meta/            # instagram.go, facebook.go, threads.go, types.go
│   │   ├── tiktok/
│   │   └── ... (10 platforms total)
│   ├── response/
│   │   └── response.go      # OK(), Fail(), HandleError(), writeJSON()
│   └── types/
│       └── types.go         # DownloadItem struct + MediaType constants
├── .env                     # DO NOT commit. Instagram cookie stored here.
└── scripts/
    └── ig-cookies.sh        # Old bash script (deprecated, replaced by cmd/ig-cookies)
```

---

## API Endpoints

All endpoints are `GET` with one required query param: `?url=<URL>`.

| Endpoint | Needs Cookie? | Description |
|---|---|---|
| `GET /health` | No | Runtime stats (uptime, memory, goroutines) |
| `GET /` | No | List all endpoints |
| `GET /api/meta/instagram/download` | Optional* | Download IG photo/video/carousel |
| `GET /api/meta/instagram/profile` | **Yes** | Scrape IG profile (followers, bio, etc.) |
| `GET /api/meta/facebook/download` | No | Download Facebook video |
| `GET /api/meta/threads/download` | No | Download Threads media |
| `GET /api/tiktok/download` | No | Download TikTok video (no watermark) |
| `GET /api/tiktok/profile` | No | Scrape TikTok profile |
| `GET /api/tiktok/music` | No | Extract audio from TikTok |
| `GET /api/twitter/download` | No | Download Twitter/X video |
| `GET /api/reddit/download` | No | Download Reddit media |
| `GET /api/pinterest/download` | No | Download Pinterest media |
| `GET /api/terabox/download` | No | Download from Terabox |
| `GET /api/spotify/download` | No | Download from Spotify |
| `GET /api/soundcloud/download` | No | Download from SoundCloud |
| `GET /api/capcut/download` | No | Download from CapCut |
| `GET /api/linkedin/download` | No | Download from LinkedIn |
| `GET /api/pixiv/download` | Optional* | Download a Pixiv artwork (illust/manga/ugoira) |
| `GET /api/pixiv/profile` | No | List a user's artworks by user URL |
| `GET /api/pixiv/illustrations` | No | List a user's illustrations |
| `GET /api/pixiv/bookmarks` | **Yes** | List a user's public bookmarks |

> *Instagram download: uses IG mobile API when cookie is present, falls back to Snapsave API otherwise.
> *Pixiv: `/download` needs a cookie only for R-18/expired works; URL fields may be null without a logged-in cookie. CDN images require `Referer: https://www.pixiv.net/`.

---

## Response Format

**Success:**
```json
{
  "status": 200,
  "success": true,
  "message": "Instagram media fetched successfully",
  "data": { ... }
}
```

**Error:**
```json
{
  "status": 400,
  "success": false,
  "message": "url parameter is required",
  "error": {
    "code": "MISSING_PARAMETER",
    "detail": "url parameter is required"
  }
}
```

**Error codes:** `MISSING_PARAMETER` (400) · `INVALID_URL` (422) · `UPSTREAM_FAILED` (502) · `NOT_FOUND` (404) · `INTERNAL_ERROR` (500)

---

## Instagram Client

Package: `internal/instagram/`

- Cookie stored as a package-level `var cookies string`, set via `SetCookies(c string)`
- `IGClient` sends all requests with mobile User-Agent (`Instagram 347.x Android`) + `X-IG-App-ID: 936619743392459`
- Cookie header set manually: `Cookie: sessionid=...; csrftoken=...`
- `X-CSRFToken` is extracted automatically from the cookie string
- Base URL: `https://i.instagram.com/api/v1`
- **Profile** → `GET /users/{username}/usernameinfo/`
- **Media** → extract shortcode from URL → scrape mediaID from IG page → `GET /media/{id}/info/`
- **Stories** → resolve username to numeric PK first → `GET /feed/user/{pk}/reel_media/`
- `HasCookies()` returns true when cookie string is non-empty AND contains `sessionid`

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `INSTAGRAM_COOKIES` | — | `sessionid=xxx; csrftoken=yyy` |
| `PIXIV_COOKIES` | — | `PHPSESSID=xxx; ...` (Pixiv login cookie; needed for R-18/bookmarks) |
| `PORT` | `3000` | Server listen port |
| `HOST` | `0.0.0.0` | Server bind address |

Loaded from `.env` at project root **or** from the shell environment. Shell env takes precedence over `.env`.
`.env` format: `KEY="value"` — outer quotes are stripped automatically.

---

## Adding a New Platform

1. Create `internal/platform/<name>/` → scraping implementation
2. Create `internal/handler/<name>.go` → thin handler, call platform func, return `response.OK()`
3. Register route in `cmd/server/routes.go`
4. Add endpoint to the list in the root `/` handler

Handler boilerplate (always the same pattern):
```go
func HandleXxxDownload(w http.ResponseWriter, r *http.Request) {
    url := r.URL.Query().Get("url")
    if url == "" {
        response.HandleError(w, errors.NewValidation("url parameter is required"))
        return
    }
    data, err := xxx.FetchData(url)
    if err != nil {
        response.HandleError(w, err)
        return
    }
    response.OK(w, data, "Xxx media fetched successfully")
}
```

---

## Key Dependencies

| Package | Used for |
|---|---|
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/chromedp/chromedp` | CDP browser automation (ig-cookies Chromium backend) |
| `github.com/mattn/go-sqlite3` | Read Firefox cookies.sqlite (CGO required) |
| `github.com/PuerkitoBio/goquery` | HTML parsing (Snapsave fallback) |
| `github.com/lmittmann/tint` | Colored slog output |

---

## Cookie Extractor CLI (`cmd/ig-cookies`)

```
ig-cookie-extractor -browser firefox
ig-cookie-extractor -browser firefox -profile ~/.config/zen/xxx.default
ig-cookie-extractor -browser chromium -exec /usr/bin/brave-browser
ig-cookie-extractor -browser firefox -out raw    # sessionid=...; csrftoken=...
ig-cookie-extractor -browser firefox -out env    # INSTAGRAM_COOKIES="..."
ig-cookie-extractor -browser firefox -out export # export INSTAGRAM_COOKIES="..."
```

**Firefox backend:** Auto-detects profile path via `filepath.Glob` with pattern `*` (not `*.default*`
because some profiles have spaces in their name, e.g. `Default (release)`).
Detection priority: `~/.config/zen/` → `~/.zen/` → `~/.mozilla/firefox/` → LibreWolf → Floorp → snap.
If multiple matches exist, picks the most recently modified one (`newestFile()`).
Copies `cookies.sqlite` + WAL files to `/tmp` before querying — safe even when browser is open.

**Chromium backend:** Launches browser with the user's own profile (auto-login). Waits for ENTER, then pulls cookies via CDP.
