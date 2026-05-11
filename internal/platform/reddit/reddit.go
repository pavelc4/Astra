package reddit

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

type Result struct {
	Platform    string `json:"platform"`
	DownloadURL string `json:"downloadUrl"`
}

func FetchData(targetURL string) (*Result, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://rapidsave.com/info?url=%s", url.QueryEscape(targetURL)), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "text/html")
	req.Header.Set("Referer", "https://rapidsave.com/")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("RapidSave request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("RapidSave response read failed")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.NewUpstream("RapidSave HTML parse failed")
	}

	href, ok := doc.Find("a.downloadbutton").Attr("href")
	if !ok || href == "" {
		return nil, errors.NewUpstream("Download link not found")
	}

	return &Result{Platform: "reddit", DownloadURL: href}, nil
}
