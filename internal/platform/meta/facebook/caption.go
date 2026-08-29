package facebook

import "strings"

// cleanFacebookCaption strips Facebook's "<stats> | <title>" og:title prefix
// (e.g. "🎬 456K views · 22K reactions | Real caption") down to the caption.
func cleanFacebookCaption(caption string) string {
	trimmed := strings.TrimSpace(caption)
	// Facebook video og:title formatting: "<Stats> | <Title>"
	// Stats might start with "🎬" (clapper board) or directly with view count text.
	if idx := strings.Index(trimmed, "|"); idx != -1 {
		prefix := strings.TrimSpace(trimmed[:idx])
		lowerPrefix := strings.ToLower(prefix)

		isStats := strings.HasPrefix(prefix, "🎬") ||
			strings.Contains(lowerPrefix, "view") ||
			strings.Contains(lowerPrefix, "react") ||
			strings.Contains(lowerPrefix, "comment") ||
			strings.Contains(lowerPrefix, "share") ||
			strings.Contains(lowerPrefix, "like") ||
			strings.Contains(lowerPrefix, "play") ||
			strings.Contains(lowerPrefix, "penayangan") ||
			strings.Contains(lowerPrefix, "tanggapan") ||
			strings.Contains(lowerPrefix, "suka") ||
			strings.Contains(lowerPrefix, "komentar") ||
			strings.Contains(lowerPrefix, "bagikan") ||
			strings.Contains(lowerPrefix, "ကြည့်ရှုမှု") ||
			strings.Contains(lowerPrefix, "တုံ့ပြန်မှု") ||
			strings.Contains(lowerPrefix, "တုံ့ပြန်ချက်") ||
			strings.Contains(lowerPrefix, "တုံပြန်မှု")

		if isStats {
			return strings.TrimSpace(trimmed[idx+1:])
		}
	}
	return caption
}
