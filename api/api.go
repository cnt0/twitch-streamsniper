package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type StreamChannel struct {
	ID                  int    `json:"_id"`
	BroadCasterLanguage string `json:"broadcaster_language"`
	CreatedAt           string `json:"created_at"`
	DisplayName         string `json:"display_name"`
	Followers           int    `json:"followers"`
	Game                string `json:"game"`
	Language            string `json:"language"`
	Logo                string `json:"logo"`
	Name                string `json:"name"`
	ProfileBanner       string `json:"profile_banner"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	URL                 string `json:"url"`
	VideoBanner         string `json:"video_banner"`
	Views               int    `json:"views"`
}

func (c *StreamChannel) CreatedAtTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05Z", c.CreatedAt)
	return t
}

func (c *StreamChannel) UpdatedAtTime() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05Z", c.UpdatedAt)
	return t
}

type StreamPreview struct {
	Large    string `json:"large"`
	Medium   string `json:"medium"`
	Small    string `json:"small"`
	Template string `json:"template"`
}

func (s *StreamPreview) UrlFromTemplate(width, height int) string {
	ret := strings.Replace(s.Template, "{width}", strconv.Itoa(width), -1)
	return strings.Replace(ret, "{height}", strconv.Itoa(height), -1)
}

type Stream struct {
	AverageFPS  float32       `json:"average_fps"`
	Channel     StreamChannel `json:"channel"`
	CreatedAt   string        `json:"created_at"`
	Delay       int           `json:"delay"`
	Game        string        `json:"game"`
	Preview     StreamPreview `json:"preview"`
	VideoHeight int           `json:"video_height"`
	Viewers     int           `json:"viewers"`
	Formats     []FormatItem  `json:"formats"`
}

type StreamResult struct {
	S *Stream
	E error
}

func NewStream(channel *StreamChannel, client, ytdl string) (*Stream, error) {
	chS := make(chan StreamResult)
	go func(id string, ch chan StreamResult) {
		req, err := http.NewRequest("GET", "https://api.twitch.tv/kraken/streams/"+id, nil)
		if err != nil {
			ch <- StreamResult{nil, err}
			return
		}
		req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
		req.Header.Set("Client-ID", client)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			ch <- StreamResult{nil, err}
			return
		}
		var newStream struct {
			S Stream `json:"stream"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&newStream); err != nil {
			ch <- StreamResult{nil, err}
			return
		}
		if err := resp.Body.Close(); err != nil {
			ch <- StreamResult{nil, err}
			return
		}
		ch <- StreamResult{&newStream.S, nil}
	}(strconv.Itoa(channel.ID), chS)
	var newFormats struct {
		Formats []FormatItem `json:"formats"`
	}
	data, err := exec.Command(ytdl, "-J", "--skip-download", channel.URL).Output()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &newFormats); err != nil {
		return nil, err
	}
	res := <-chS
	if res.E != nil {
		return nil, err
	}
	res.S.Formats = newFormats.Formats
	return res.S, nil
}

type FormatItem struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}

type FollowedStreams struct {
	Total   int      `json:"_total"`
	Streams []Stream `json:"streams"`
}

func (f *FollowedStreams) UpdateStream(channelName, client, ytdl string) (*Stream, error) {
	for i, s := range f.Streams {
		if s.Channel.DisplayName == channelName {
			if s1, err := NewStream(&s.Channel, client, ytdl); err != nil {
				return nil, err
			} else {
				f.Streams[i] = *s1
			}
			return &f.Streams[i], nil
		}
	}
	return nil, errors.New(channelName + ": no such channel")
}

func ParseFollowedStreams(client, auth string) (*FollowedStreams, error) {
	req, err := http.NewRequest("GET", "https://api.twitch.tv/kraken/streams/followed", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Set("Client-ID", client)
	req.Header.Set("Authorization", "OAuth "+auth)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var followedStreams FollowedStreams
	if err := json.NewDecoder(resp.Body).Decode(&followedStreams); err != nil {
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}
	// var wg sync.WaitGroup
	// wg.Add(followedStreams.Total)
	// for _, s := range followedStreams.Streams {
	// 	go func(s *Stream) {
	// 		if err := s.UpdateFormats(); err != nil {
	// 			log.Println(err)
	// 		}
	// 		wg.Done()
	// 	}(&s)
	// }
	// wg.Wait()
	return &followedStreams, nil
}
