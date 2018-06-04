package connection

import "fmt"

// Kekeke ..
func Kekeke() {
	fmt.Println("le;e;e")
}

// PlayerConnection ...
type PlayerConnection interface {
	PlayVideo(addr string) error
}
