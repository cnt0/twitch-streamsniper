package mpris

import (
	"errors"
	"strings"

	"github.com/godbus/dbus"
)

const (
	mprisPath      = "org.mpris.MediaPlayer2"
	mprisInterface = "/org/mpris/MediaPlayer2"
)

func upgradeToMRL(url string) string {
	if strings.HasPrefix(url, "http") {
		return url
	}
	return "file://" + url
}

// Connection ...
type Connection struct {
	conn *dbus.Conn
}

// NewConnection ...
func NewConnection() (*Connection, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	return &Connection{conn}, nil
}

func (c *Connection) playerInstance() (string, error) {
	ret := c.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0)
	if ret.Err != nil {
		return "", ret.Err
	}
	if len(ret.Body) == 0 {
		return "", errors.New("DBus connection error")
	}
	for _, s := range ret.Body[0].([]string) {
		if strings.HasPrefix(s, mprisPath) {
			return s, nil
		}
	}
	return "", errors.New("no mpris player instance found")
}

// PlayVideo ...
func (c *Connection) PlayVideo(addr string) error {
	instanceName, err := c.playerInstance()
	if err != nil {
		return err
	}
	call := c.conn.
		Object(instanceName, mprisInterface).
		Call("org.mpris.MediaPlayer2.Player.OpenUri", 0, upgradeToMRL(addr))
	return call.Err
}
