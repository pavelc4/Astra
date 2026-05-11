package httpclient

import (
	"net/http"
	"time"
)

var Client = &http.Client{
	Timeout: 15 * time.Second,
}

var DefaultHeaders = http.Header{
	"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
}
