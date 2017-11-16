package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/mpv"
	"github.com/cnt0/twitch-streamsniper/api"
)

const ytdl = "youtube-dl"

var yt = flag.String("yt", "", "")

func main() {
	flag.Parse()
	c := mpv.NewClient(mpv.NewIPCClient("/tmp/mpvsocket"))
	if len(*yt) > 0 {
		c.SetProperty("ytdl-format", *yt)
	}

	if u, err := url.Parse(os.Args[len(os.Args)-1]); err == nil {
		if strings.HasSuffix(strings.ToLower(u.Hostname()), "twitch.tv") {
			var formats struct {
				Formats []api.FormatItem `json:"formats"`
			}
			data, err := exec.Command(ytdl, "-J", "--skip-download", os.Args[len(os.Args)-1]).Output()
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := json.Unmarshal(data, &formats); err != nil {
				fmt.Println(err)
				return
			}

			for _, f := range formats.Formats {
				s := strings.ToLower(f.Format)
				if strings.Contains(s, "source") || strings.Contains(s, "1080p60") {
					c.Loadfile(f.URL, mpv.LoadFileModeReplace)
					return
				}
			}
			fmt.Println("Hmmm... What does hero truly need?")
			for i, f := range formats.Formats {
				fmt.Printf("%v: %v\n", i+1, f.Format)
			}
			var idx int
			fmt.Scan(&idx)
			c.Loadfile(formats.Formats[idx-1].URL, mpv.LoadFileModeReplace)
			return
		}
	}

	if len(os.Args) > 1 {
		c.Loadfile(os.Args[len(os.Args)-1], mpv.LoadFileModeReplace)
	}

}
