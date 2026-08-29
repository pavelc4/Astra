package facebook

import "testing"

func TestCleanFacebookCaption(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "🎬 ကြည့်ရှုမှု ၆.၁ ထောင် ကြိမ် · တုံပြန်မှု ၅.၉ ထောင် ခု | emang iya ya cosplayer mukanya dempul banget?",
			expected: "emang iya ya cosplayer mukanya dempul banget?",
		},
		{
			input:    "ကြည့်ရှုမှု ၁.၂ သန်း ကြိမ် · တုံ့ပြန်မှု ၅.၄ ထောင် ခု | Untung ketahuan\nHati hati buat para lelaki | Bagja Gumilar",
			expected: "Untung ketahuan\nHati hati buat para lelaki | Bagja Gumilar",
		},
		{
			input:    "🎬 456K views · 22K reactions | The actual caption here",
			expected: "The actual caption here",
		},
		{
			input:    "456K views · 22K reactions | The actual caption here",
			expected: "The actual caption here",
		},
		{
			input:    "🎬 Views 1.2M | Title with a | character | inside",
			expected: "Title with a | character | inside",
		},
		{
			input:    "Views 1.2M | Title with a | character | inside",
			expected: "Title with a | character | inside",
		},
		{
			input:    "No emoji at start, just normal caption",
			expected: "No emoji at start, just normal caption",
		},
		{
			input:    "🎬 No pipe symbol in this one",
			expected: "🎬 No pipe symbol in this one",
		},
		{
			input:    "   🎬 Spaces before emoji | Clean me   ",
			expected: "Clean me",
		},
		{
			input:    "Spaces before text | Clean me   ",
			expected: "Spaces before text | Clean me   ", // No stats keyword, so it shouldn't clean. Correct!
		},
	}

	for _, tt := range tests {
		result := cleanFacebookCaption(tt.input)
		if result != tt.expected {
			t.Errorf("cleanFacebookCaption(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}
