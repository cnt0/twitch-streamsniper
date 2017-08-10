package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"runtime"
	"sync"

	"github.com/blang/mpv"
	"github.com/rakyll/statik/fs"
	"github.com/rs/cors"

	"github.com/cnt0/twitch-streamsniper/api"
	_ "github.com/cnt0/twitch-streamsniper/site/statik"
)

var (
	m       sync.Mutex
	streams *api.FollowedStreams
	client  = flag.String("client", "", "")
	auth    = flag.String("auth", "", "")
	ytdl    = flag.String("ytdl", "", "")
	socket  = flag.String("socket", "", "")
)

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

func HandleUpdateFormats(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("s")
	log.Println("update formats for " + channel)
	if channel == "" {
		m.Lock()
		if err := json.NewEncoder(w).Encode(streams); err != nil {
			log.Println(err)
		}
		m.Unlock()
	}
	stream, err := streams.UpdateFormats(channel, *ytdl)
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

func HandlePlayVideo(w http.ResponseWriter, r *http.Request) {
	if len(*socket) > 0 {
		var s struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			log.Println(err)
		}
		log.Printf("playing video %v", s.URL)
		mpv.NewClient(mpv.NewIPCClient(*socket)).Loadfile(s.URL, mpv.LoadFileModeReplace)
	}

}

func init() {
	flag.Parse()
	if len(*ytdl) == 0 {
		*ytdl = "youtube-dl"
	}
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
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
	mux.HandleFunc("/formats", HandleUpdateFormats)
	mux.HandleFunc("/update", HandleUpdateAll)
	mux.HandleFunc("/play", HandlePlayVideo)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:8080",
			//"http://localhost:4200",
		},
		AllowCredentials: true,
	})
	http.ListenAndServe(":8080", c.Handler(mux))
}
