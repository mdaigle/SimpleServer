package main

import(
	"github.com/mdaigle/SimpleServer/protocol"
	"fmt"
	"net"
	"os"
	"time"
	"sync"
	"bufio"
	"io"
)

//TODO: error checking

func main() {
	portnum := os.Args[1]
	udpAddr, err := net.ResolveUDPAddr("up4", portnum)
	var wg sync.WaitGroup
	quit := make(chan bool)
	end := make(chan bool)

	go broadcastQuit(quit, end)
	go readIn(end)

	for {
		conn, err := net.ListenUDP("udp", udpAddr)
		if err == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				handleClient(conn, quit, end)
			}()
		} else {
			//There was an error, so quit the server
			end <- true;
			break;
		}
	}
	wg.Wait()
}

// Closes quit when correct input is detected. Closes quit if notified to do so through end channel
// (for reporting of errors by other routines).
func broadcastQuit(quit *chan bool, end *chan bool) {
	for {
		select {
		case endserver := <-end:
			if endserver {
				close(quit)
				break
			}
		}
	}
}

// Waits for input on stdin. Sends end message if 'q' or EOF detected.
func readIn(end *chan bool) {
	reader := bufio.NewReader(os.Stdin)

	for {
		text, err := reader.ReadString('\n')

		if text == "q" {
			//post to channel or global
			end <- true;
			break;
		}

		if err == io.EOF {
			end <- true;
			break;
		}
	}
}

func handleClient(conn *net.UDPConn, quit *chan bool, end *chan bool) {
	fmt.Println("Session created")

	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	for {
		// use select to see if we get a notification that we are quitting
		var buf [512]byte
		_, addr, err := conn.ReadFromUDP(buf[0:])

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				// we timed out
				break;
			}
			//TODO: handle other errors
		}

		message := protocol.Decode(buf)

		select {
		case _,ok := <- quit:
			if !ok {
				break;
			}
		}
		// logic for state transitions
	}

	//send goodbye message
	err := conn.Close()
	if err != nil {
		end <- true
	}
}