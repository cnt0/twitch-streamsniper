package mpris

import (
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus"
)

const mprisInterface = "/org/mpris/MediaPlayer2"

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
	obj := c.conn.BusObject()
	fmt.Println(obj.Path())
	fmt.Println(obj.Destination())
	ret := c.conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0)
	if ret.Err != nil {
		return "", ret.Err
	}
	if len(ret.Body) == 0 {
		return "", errors.New("DBus connection error")
	}
	for _, s := range ret.Body[0].([]string) {
		if strings.HasPrefix(s, "org.mpris.MediaPlayer2") {
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
	//fmt.Println(instanceName)
	obj := c.conn.Object(instanceName, "/org/mpris/MediaPlayer2")
	call := obj.Call("org.mpris.MediaPlayer2.Player.OpenUri", 0, upgradeToMRL(addr))
	if call.Err != nil {
		return call.Err
	}
	return nil
}
