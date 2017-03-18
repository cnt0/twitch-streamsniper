package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"runtime"
	"sync"

	"github.com/rakyll/statik/fs"

	"github.com/cnt0/twitch-streamsniper/api"
	_ "github.com/cnt0/twitch-streamsniper/site/statik"
)

var (
	m       sync.Mutex
	streams *api.FollowedStreams
	client  = flag.String("client", "", "")
	auth    = flag.String("auth", "", "")
	ytdl    = flag.String("ytdl", "", "")
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
	if channel == "" {
		m.Lock()
		if err := json.NewEncoder(w).Encode(streams); err != nil {
			log.Println(err)
		}
		m.Unlock()
	}
	m.Lock()
	stream, err := streams.UpdateFormats(channel, *ytdl)
	m.Unlock()
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
	http.Handle("/", http.FileServer(statikFS))
	http.HandleFunc("/formats", HandleUpdateFormats)
	http.HandleFunc("/update", HandleUpdateAll)

	http.ListenAndServe(":8080", nil)
}
