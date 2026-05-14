package extractor

// InstagramCookies holds the extracted Instagram session cookies.
type InstagramCookies struct {
	SessionID string
	CSRFToken string
}

// Extractor is the interface implemented by each browser backend.
type Extractor interface {
	Extract() (*InstagramCookies, error)
}
