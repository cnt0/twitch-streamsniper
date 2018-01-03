package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/mpv"
	"github.com/cnt0/twitch-streamsniper/api"
)

const (
	ytdl      = "youtube-dl"
	mpvSocket = "/tmp/mpvsocket"
)

func main() {
	c := mpv.NewClient(mpv.NewIPCClient(mpvSocket))

	if u, err := url.Parse(os.Args[len(os.Args)-1]); err == nil {

		hostname := strings.ToLower(u.Hostname())
		var formats struct {
			Formats []api.FormatItem `json:"formats"`
		}

		if strings.HasSuffix(hostname, "twitch.tv") {
			data, err := exec.Command(ytdl, "-J", "--skip-download", os.Args[len(os.Args)-1]).Output()
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := json.Unmarshal(data, &formats); err != nil {
				fmt.Printf("%v is likely offline\n", u.Path)
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

		if strings.HasSuffix(hostname, "youtube.com") || strings.HasSuffix(hostname, "youtu.be") {
			data, err := exec.Command(ytdl, "-J", "--skip-download", os.Args[len(os.Args)-1]).Output()
			if err != nil {
				fmt.Println(err)
				return
			}
			if err := json.Unmarshal(data, &formats); err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Hmmm... What does hero truly need?")
			for i, f := range formats.Formats {
				fmt.Printf("%v: %v\n", i+1, f.Format)
			}
			desiredFormat := ""
			fmt.Scan(&desiredFormat)
			c.SetProperty("ytdl-format", desiredFormat)
			c.Loadfile(os.Args[len(os.Args)-1], mpv.LoadFileModeReplace)
			return
		}

	}

}
