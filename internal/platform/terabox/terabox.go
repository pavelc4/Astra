package terabox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

type Result struct {
	Platform string          `json:"platform"`
	Raw      json.RawMessage `json:"raw"`
}

var nonceRe = regexp.MustCompile(`"nonce":"(.*?)"`)

func FetchData(ctx context.Context, teraboxURL string) (*Result, error) {
	nonce, err := fetchNonce(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"action": {"terabox_fetch"},
		"url":    {teraboxURL},
		"nonce":  {nonce},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://teradownloaderz.com/wp-admin/admin-ajax.php", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://teradownloaderz.com")
	req.Header.Set("Referer", "https://teradownloaderz.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Terabox fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("Terabox response read failed")
	}

	return &Result{Platform: "terabox", Raw: json.RawMessage(body)}, nil
}

func fetchNonce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://teradownloaderz.com", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", errors.NewUpstream(fmt.Sprintf("Terabox nonce fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.NewUpstream("Terabox nonce HTML parse failed")
	}

	script := doc.Find("#jquery-core-js-extra").Text()
	if script == "" {
		return "", errors.NewUpstream("Nonce script not found")
	}

	m := nonceRe.FindStringSubmatch(script)
	if len(m) < 2 {
		return "", errors.NewUpstream("Nonce not found")
	}

	return m[1], nil
}
