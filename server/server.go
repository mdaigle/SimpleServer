package main

import(
	"github.com/mdaigle/SimpleServer/protocol"
	"fmt"
	"net"
	"os"
	"time"
	"sync"
)

func main() {
	portnum := os.Args[1]
	udpAddr, err := net.ResolveUDPAddr("up4", portnum)
	var wg sync.WaitGroup

	for {
		// do select to listen for quit command "q"
		// when we hear it, post to a shared channel that we're quitting
		conn, err := net.ListenUDP("udp", udpAddr)
		wg.Add(1)
		go func() {
			handleClient(conn)
			wg.Done()
		}()
	}
	wg.Wait()
}

func handleClient(conn *net.UDPConn) {
	fmt.Println("Session created")

	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	for {
		// use select to see if we get a notification that we are quitting
		var buf [512]byte
		_, addr, err := conn.ReadFromUDP(buf[0:])

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				// we timed out, so send a goodbye message and close the connection
			}
			// something else if messed up
		}

		message := protocol.Decode(buf);
		// logic for state transitions
	}
}