package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
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

func HandleStreamUrls(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Query().Get("q"))
	// if formats, err := StreamUrls(r.URL.Query().Get("q")); err != nil {
	// 	log.Println(err)
	// } else {
	// 	log.Println("got formats: ", len(formats))
	// 	w.Header().Add("Access-Control-Allow-Origin", "*")
	// 	if err := json.NewEncoder(w).Encode(formats); err != nil {
	// 		log.Println(err)
	// 	}
	// }
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
	stream, err := streams.UpdateFormats(channel)
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

func RunServer() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)
	fs := http.FileServer(http.Dir(path.Join(dir, "static")))
	http.Handle("/", fs)
	http.HandleFunc("/stream", HandleStreamUrls)
	http.ListenAndServe(":3000", nil)
}

func init() {
	flag.Parse()
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("client: %v", *client)
	log.Printf("auth: %v", *auth)

	// followedStreams, err := api.ParseFollowedStreams()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, s := range followedStreams.Streams {
	// 	fmt.Println(s.Channel.DisplayName, s.Channel.Status)
	// 	for _, f := range s.Formats {
	// 		fmt.Println(f.Format)
	// 	}
	// 	fmt.Println()
	// }

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
