package handler

import (
	"github.com/pavelc4/astra/internal/platform/spotify"
)

var HandleSpotifyDownload = makeDownloadHandler(spotify.FetchData, "Spotify media fetched successfully")
