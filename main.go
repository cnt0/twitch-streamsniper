package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/rakyll/statik/fs"
	"github.com/rs/cors"

	"github.com/cnt0/golang-http-utils/utils"
	"github.com/cnt0/twitch-streamsniper/api"
	"github.com/cnt0/twitch-streamsniper/player-connection/mpris"
	"github.com/cnt0/twitch-streamsniper/player-connection/mpv"

	_ "github.com/cnt0/twitch-streamsniper/site/statik"
)

const handlerTimeout = 5 * time.Second

var (
	m       sync.Mutex
	streams *api.FollowedStreams
	client  = flag.String("client", "", "client field for twitch authentication")
	auth    = flag.String("auth", "", "auth field for twitch authentication")
	ytdl    = flag.String("ytdl", "youtube-dl", "path to youtube-dl executable")
	socket  = flag.String("socket", "/tmp/mpvsocket", "path to unix socket for communication with mpv")
	isMPV   = flag.Bool("mpv", false, "connect to mpv socket instead of mpris dbus")

	mpvConnection   *mpv.Connection
	mprisConnection *mpris.Connection
)

func playVideo(addr string) error {
	if *isMPV {
		return mpvConnection.PlayVideo(addr)
	}
	return mprisConnection.PlayVideo(addr)
}

// HandleUpdateAll ...
func HandleUpdateAll(w http.ResponseWriter, r *http.Request) {
	m.Lock()
	var err error
	if streams, err = api.ParseFollowedStreams(*client, *auth); err != nil {
		log.Println(err)
	}
	m.Unlock()
	if err := json.NewEncoder(w).Encode(streams); err != nil {
		log.Println(err)
	}
}

// HandleUpdateFormats ...
func HandleUpdateFormats(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("s")
	log.Println("update stream for " + channel)
	if channel == "" {
		m.Lock()
		if err := json.NewEncoder(w).Encode(streams); err != nil {
			log.Println(err)
		}
		m.Unlock()
	}
	stream, err := streams.UpdateStream(channel, *client, *ytdl)
	if err != nil {
		log.Println(err)
	} else if stream == nil {
		log.Printf("%s: no such stream!", channel)
	} else {
		if err := json.NewEncoder(w).Encode(stream); err != nil {
			log.Println(err)
		}
	}
}

// HandlePlayVideo ...
func HandlePlayVideo(w http.ResponseWriter, r *http.Request) {
	log.Printf("request: %v\n", r.Method)
	if len(*socket) > 0 && r.Method == "POST" {
		var s struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			log.Println(err)
		}
		log.Printf("playing video %v", s.URL)
		if err := playVideo(s.URL); err != nil {
			log.Println(err)
		}
	}

}

func init() {
	flag.Parse()
}

func main() {

	runtime.GOMAXPROCS(1)
	log.Printf("client: %v", *client)
	log.Printf("auth: %v", *auth)

	m.Lock()
	var err error
	if streams, err = api.ParseFollowedStreams(*client, *auth); err != nil {
		log.Fatal(err)
	}
	m.Unlock()

	statikFS, _ := fs.New()
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(statikFS))
	mux.Handle("/formats", http.TimeoutHandler(http.HandlerFunc(HandleUpdateFormats), handlerTimeout, "timeout in formats handler"))
	mux.Handle("/update", http.TimeoutHandler(http.HandlerFunc(HandleUpdateAll), handlerTimeout, "timeout in update handler"))
	mux.Handle("/play", http.TimeoutHandler(http.HandlerFunc(HandlePlayVideo), handlerTimeout, "timeout in play handler"))
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:8080",
			"http://localhost:4200",
		},
		AllowCredentials: true,
	})
	utils.ListenAndServeSA(":8080", c.Handler(mux))

}
