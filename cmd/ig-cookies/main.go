package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pavelc4/astra/cmd/ig-cookies/extractor"
)

const usage = `ig-cookie-extractor — extract Instagram sessionid + csrftoken from your browser

Usage:
  ig-cookie-extractor -browser <firefox|chromium> [options]

Options:
  -browser  firefox|chromium   Browser backend (required)
  -profile  /path/to/profile   Override profile location (optional, auto-detect by default)
  -exec     /path/to/binary    Override browser binary (optional, chromium only)
  -out      export|env|raw     Output format (default: export)

Example:
  ig-cookie-extractor -browser firefox
  ig-cookie-extractor -browser firefox -profile ~/.zen/abc123.default-release
  ig-cookie-extractor -browser chromium -exec /usr/bin/brave-browser
  ig-cookie-extractor -browser chromium -out raw
`

func main() {
	var (
		browserFlag = flag.String("browser", "", "Browser backend: firefox | chromium")
		profileFlag = flag.String("profile", "", "Path to browser profile")
		execFlag    = flag.String("exec", "", "Path to browser binary")
		outFlag     = flag.String("out", "export", "Output format: export | env | raw")
	)

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if *browserFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: the -browser flag is required (firefox or chromium)")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	fmt.Println("=== Instagram Cookie Extractor ===")
	fmt.Println()

	var ex extractor.Extractor

	switch *browserFlag {
	case "firefox":
		// NewFirefox does not return an error —
		// the profile is resolved lazily during Extract().
		ex = extractor.NewFirefox(*profileFlag)

	case "chromium":
		var err error
		ex, err = extractor.NewChromiumExtractor(*profileFlag, *execFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(
			os.Stderr,
			"Error: unknown browser '%s'. Use 'firefox' or 'chromium'\n",
			*browserFlag,
		)
		os.Exit(1)
	}

	cookies, err := ex.Extract()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	cookieStr := fmt.Sprintf(
		"sessionid=%s; csrftoken=%s",
		cookies.SessionID,
		cookies.CSRFToken,
	)

	fmt.Println()
	fmt.Println("Cookies extracted successfully! ")
	fmt.Println()

	switch *outFlag {
	case "export":
		fmt.Printf("export INSTAGRAM_COOKIES=\"%s\"\n", cookieStr)

	case "env":
		fmt.Printf("INSTAGRAM_COOKIES=\"%s\"\n", cookieStr)

	case "raw":
		fmt.Println(cookieStr)

	default:
		fmt.Fprintf(
			os.Stderr,
			"Warning: unknown format '%s', falling back to 'export'\n\n",
			*outFlag,
		)
		fmt.Printf("export INSTAGRAM_COOKIES=\"%s\"\n", cookieStr)
	}
}
