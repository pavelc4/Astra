package handler

import (
	"github.com/pavelc4/astra/internal/platform/soundcloud"
)

var HandleSoundcloudDownload = makeDownloadHandler(soundcloud.FetchData, "SoundCloud media fetched successfully")
