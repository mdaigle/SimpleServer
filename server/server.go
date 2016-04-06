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
		//Check if we should quit
		select{
		case _,ok := <- quit:
			if !ok {
				break;
			}
		}

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

// Closes quit when a shutdown command is received through the end channel.
func broadcastQuit(quit chan<- bool, end <-chan bool) {
	<-end
	close(quit)
}

// Waits for input on stdin. Sends shutdown command if 'q' or EOF detected.
func readIn(end chan<- bool) {
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

func handleClient(conn *net.UDPConn, quit <-chan bool, end chan<- bool) {
	fmt.Println("Session created")

	conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	for {
		//read on a separate routine and notify through channel to prevent blocking here?
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
		//TODO: transmit message contents back to client


		select {
		case _,ok := <- quit:
			if !ok {
				break;
			}
		}
		//TODO: logic for state transitions
	}

	//TODO: send goodbye message
	err := conn.Close()
	if err != nil {
		end <- true
	}
}