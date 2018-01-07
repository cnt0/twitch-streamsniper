package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/blang/mpv"
	"github.com/rakyll/statik/fs"
	"github.com/rs/cors"

	"github.com/cnt0/twitch-streamsniper/api"
	_ "github.com/cnt0/twitch-streamsniper/site/statik"
)

const handlerTimeout = 5 * time.Second

var (
	m       sync.Mutex
	streams *api.FollowedStreams
	client  = flag.String("client", "", "")
	auth    = flag.String("auth", "", "")
	ytdl    = flag.String("ytdl", "", "")
	socket  = flag.String("socket", "", "")

	mpvClientMutex sync.Mutex
	mpvClient      *mpv.Client
)

func statTime(name string) (ctime time.Time, err error) {
	fi, err := os.Stat(name)
	if err != nil {
		return
	}
	stat := fi.Sys().(*syscall.Stat_t)
	ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	return
}

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

func HandlePlayVideo(w http.ResponseWriter, r *http.Request) {
	if len(*socket) > 0 {
		var s struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			log.Println(err)
		}
		log.Printf("playing video %v", s.URL)
		mpvClientMutex.Lock()
		if mpvClient == nil {
			mpvClient = mpv.NewClient(mpv.NewIPCClient(*socket))
		}
		if err := mpvClient.Loadfile(s.URL, mpv.LoadFileModeReplace); err != nil {
			log.Println(err)
		}
		mpvClientMutex.Unlock()
	}

}

func init() {
	flag.Parse()
	if len(*ytdl) == 0 {
		*ytdl = "youtube-dl"
	}
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
	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()) {
		if l, err := net.FileListener(os.NewFile(3, "socket")); err != nil {
			fmt.Println(err)
		} else {
			if err := http.Serve(l, c.Handler(mux)); err != nil {
				fmt.Println(err)
			}
		}
	} else {
		http.ListenAndServe(":8080", c.Handler(mux))
	}
}
