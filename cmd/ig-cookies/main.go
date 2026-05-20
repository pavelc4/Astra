package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pavelc4/astra/cmd/ig-cookies/extractor"
)

const usage = `ig-cookie-extractor — extract Instagram / Facebook cookies from your browser

Usage:
  ig-cookie-extractor -browser <firefox|chromium> -platform <instagram|facebook> [options]

Options:
  -browser  firefox|chromium          Browser backend (required)
  -platform instagram|facebook         Platform (default: instagram)
  -profile  /path/to/profile          Override profile location (optional)
  -exec     /path/to/binary           Override browser binary (optional, chromium only)
  -out      export|env|raw            Output format (default: export)

Example:
  ig-cookie-extractor -browser firefox
  ig-cookie-extractor -browser firefox -platform facebook
  ig-cookie-extractor -browser chromium -exec /usr/bin/brave-browser -out raw
`

func main() {
	var (
		browserFlag  = flag.String("browser", "", "Browser backend: firefox | chromium")
		platformFlag = flag.String("platform", "instagram", "Platform: instagram | facebook")
		profileFlag  = flag.String("profile", "", "Path to browser profile")
		execFlag     = flag.String("exec", "", "Path to browser binary")
		outFlag      = flag.String("out", "export", "Output format: export | env | raw")
	)

	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if *browserFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: the -browser flag is required (firefox or chromium)")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	platform := *platformFlag
	if platform != "instagram" && platform != "facebook" {
		fmt.Fprintf(os.Stderr, "Error: unknown platform '%s'. Use 'instagram' or 'facebook'\n", platform)
		os.Exit(1)
	}

	label := "Instagram"
	envVar := "INSTAGRAM_COOKIES"
	if platform == "facebook" {
		label = "Facebook"
		envVar = "FACEBOOK_COOKIES"
	}

	fmt.Printf("=== %s Cookie Extractor ===\n", label)
	fmt.Println()

	var ex extractor.Extractor

	switch *browserFlag {
	case "firefox":
		ex = extractor.NewFirefox(*profileFlag, platform)

	case "chromium":
		var err error
		ex, err = extractor.NewChromiumExtractor(*profileFlag, *execFlag, platform)
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

	cookieStr := formatCookieString(cookies, platform)

	fmt.Println()
	fmt.Println("Cookies extracted successfully! ")
	fmt.Println()

	switch *outFlag {
	case "export":
		fmt.Printf("export %s=\"%s\"\n", envVar, cookieStr)

	case "env":
		fmt.Printf("%s=\"%s\"\n", envVar, cookieStr)

	case "raw":
		fmt.Println(cookieStr)

	default:
		fmt.Fprintf(
			os.Stderr,
			"Warning: unknown format '%s', falling back to 'export'\n\n",
			*outFlag,
		)
		fmt.Printf("export %s=\"%s\"\n", envVar, cookieStr)
	}
}

func formatCookieString(c *extractor.Cookies, platform string) string {
	var parts []string
	switch platform {
	case "facebook":
		for _, name := range []string{"c_user", "xs", "fr", "dpr"} {
			if v, ok := c.Values[name]; ok {
				parts = append(parts, name+"="+v)
			}
		}
	default:
		if v, ok := c.Values["sessionid"]; ok {
			parts = append(parts, "sessionid="+v)
		}
		if v, ok := c.Values["csrftoken"]; ok {
			parts = append(parts, "csrftoken="+v)
		}
	}
	return strings.Join(parts, "; ")
}
