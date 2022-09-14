package my_vsock

import (
	"testing"
)

func TestStart(t *testing.T) {
	go StartServer(3000)
	go func() {
		for {
			msg := <-RECV_MSG_CHAN
			t.Logf("Message from server chan: %s", string(msg.Data))
		}
	}()
	go ConnetctServer(2, 3000)
	// go func() {
	for {
		msg := <-MSG_FROM_SERVER_CHAN
		t.Logf("Message from server chan: %s", string(msg.Data))
	}
	// }()
}
