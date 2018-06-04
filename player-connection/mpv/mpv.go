package mpv

import blang "github.com/blang/mpv"

// Connection ...
type Connection struct {
	mpvClient *blang.Client
}

// NewConnection ...
func NewConnection(socket string) *Connection {
	return &Connection{
		mpvClient: blang.NewClient(blang.NewIPCClient(socket)),
	}
}

// PlayVideo ...
func (c *Connection) PlayVideo(addr string) error {
	return c.mpvClient.Loadfile(addr, blang.LoadFileModeReplace)
}
