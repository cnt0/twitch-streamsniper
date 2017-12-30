package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/esiqveland/notify"
	"github.com/godbus/dbus"

	"cloud.google.com/go/firestore"
)

const (
	TwitchID            = "TWITCH_ID"
	TwitchAuth          = "TWITCH_AUTH"
	FirestoreApp        = "TWITCH_FIRESTORE_APP"
	FirestoreCollection = "FIRESTORE_COLLECTION"
	FirestoreDocID      = "FIRESTORE_DOC_ID"
)

type TwitchChannel struct {
	DisplayName string `json:"display_name"`
	Game        string `json:"game"`
	Status      string `json:"status"`
}

type TwitchData struct {
	LastModified time.Time
	Channels     map[string]*TwitchChannel
}

type TwitchDataWithErr struct {
	*TwitchData
	error
}

var TwitchAuthHeaders = http.Header{
	"Accept":        []string{"application/vnd.twitchtv.v5+json"},
	"Client-ID":     []string{},
	"Authorization": []string{"OAuth "},
}

const TwitchAPIFollowed = "https://api.twitch.tv/kraken/streams/followed"

func init() {
	TwitchAuthHeaders.Set("Client-ID", os.Getenv(TwitchID))
	TwitchAuthHeaders.Set("Authorization",
		TwitchAuthHeaders.Get("Authorization")+os.Getenv(TwitchAuth))
}

func OldChannels(client *firestore.Client, ctx context.Context) TwitchDataWithErr {

	snapshot, err := client.Collection(os.Getenv(FirestoreCollection)).
		Doc(os.Getenv(FirestoreDocID)).
		Get(ctx)
	if err != nil {
		return TwitchDataWithErr{nil, err}
	}
	var doc TwitchData
	if err := snapshot.DataTo(&doc); err != nil {
		return TwitchDataWithErr{nil, err}
	}
	return TwitchDataWithErr{&doc, nil}
}

func NewChannels() TwitchDataWithErr {
	req, err := http.NewRequest("GET", TwitchAPIFollowed, nil)
	if err != nil {
		return TwitchDataWithErr{nil, err}
	}
	req.Header = TwitchAuthHeaders
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TwitchDataWithErr{nil, err}
	}
	defer resp.Body.Close()
	var data struct {
		Streams []struct {
			Channel struct {
				Name string `json:"name"`
				*TwitchChannel
			} `json:"channel"`
		} `json:"streams"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return TwitchDataWithErr{nil, err}
	}
	ret := TwitchData{
		LastModified: time.Now(),
		Channels:     make(map[string]*TwitchChannel),
	}
	for _, stream := range data.Streams {
		ret.Channels[stream.Channel.Name] = stream.Channel.TwitchChannel
	}
	return TwitchDataWithErr{&ret, nil}
}

func equal(data1, data2 *TwitchChannel) bool {
	return data1.DisplayName == data2.DisplayName &&
		data1.Game == data2.Game &&
		data1.Status == data2.Status
}

func main() {

	// initialize dbus connection
	conn, err := dbus.SessionBus()
	if err != nil {
		fmt.Println(err)
		return
	}
	notifier, err := notify.New(conn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer notifier.Close()

	// initialize firestore connection
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, os.Getenv(FirestoreApp))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	// execute requests in parallel
	var newChannelsRes TwitchDataWithErr
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// get fresh channels from twitch
		newChannelsRes = NewChannels()
		wg.Done()
	}()
	// get old channels from firestore
	oldChannelsRes := OldChannels(client, ctx)
	if oldChannelsRes.error != nil {
		fmt.Println(oldChannelsRes.error)
		return
	}
	wg.Wait()
	if newChannelsRes.error != nil {
		fmt.Println(newChannelsRes.error)
		return
	}
	// handle offline channels
	// (in old but not in new -> offline)
	for ch, data := range oldChannelsRes.Channels {
		if _, ok := newChannelsRes.Channels[ch]; !ok {
			if _, err := notifier.SendNotification(notify.Notification{
				Summary: data.DisplayName,
				Body:    "is no longer online",
			}); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	// handle new channels and updated channel headers
	for ch, dataNew := range newChannelsRes.Channels {
		dataOld, ok := oldChannelsRes.Channels[ch]
		if !ok || !equal(dataNew, dataOld) {
			if _, err := notifier.SendNotification(notify.Notification{
				Summary: fmt.Sprintf("%v is playing %v", dataNew.DisplayName, dataNew.Game),
				Body:    dataNew.Status,
			}); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	_, err = client.Collection(os.Getenv(FirestoreCollection)).
		Doc(os.Getenv(FirestoreDocID)).
		Set(ctx, newChannelsRes.TwitchData)
	if err != nil {
		fmt.Println(err)
		return
	}

}
