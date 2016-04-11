package main

import(
	"github.com/mdaigle/SimpleServer/protocol"
	"fmt"
	"net"
	"os"
	"time"
	//"sync"
	"bufio"
	"io"
	"sync"
)

// Global mapping from session ids to channels for session threads
var chanmap = make(map[uint32](chan protocol.P0Pmessage))
var chanmaplock sync.Mutex

// A wait group incremented for each client thread opened.
// Decremented when client threads die.
var sessionWaitGroup sync.WaitGroup

// When closed, signals that threads should end execution
var quit = make(chan bool)
// A write to this channel signals that the server should shut down
var end = make(chan bool)

func main() {
	portnum := os.Args[1];
	udpaddr, err := net.ResolveUDPAddr("udp", "localhost:"+portnum)
	fmt.Println(udpaddr.String())
	if err != nil {
		fmt.Println("error resolving address")
	}

	go broadcastQuit()
	go readIn()

	fmt.Println("listening for connections on port ", portnum)
	conn, err := net.ListenUDP("udp", udpaddr)

	for {
		//Check if we should quit
		select{
		case _,ok := <-quit:
			if !ok {
				break;
			}
		default:

		}

		fmt.Println("About to read from udp")
		buf := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
		_, addr, err := conn.ReadFromUDP(buf[0:])


		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				fmt.Println("Read timed out")
				continue
			}
			fmt.Println("Encountered an error while listening for connections")
			fmt.Println(err.Error())
			end <- true
			break
		}
		fmt.Println("New message from ", addr.String())

		sessionWaitGroup.Add(1)
		go func() {
			defer sessionWaitGroup.Done()
			processPacket(conn, buf)
		}()
	}
	sessionWaitGroup.Wait()
	conn.Close()
}

// Closes quit when a shutdown command is received through the end channel.
func broadcastQuit() {
	<-end
	fmt.Println("End request received")
	close(quit)
}

// Waits for input on stdin. Sends shutdown command if 'q' or EOF detected.
func readIn() {
	fmt.Println("In input routine.")
	reader := bufio.NewReader(os.Stdin)

	for {
		text, err := reader.ReadString('\n')
		fmt.Println(text)

		if text == "q" {
			//post to channel or global
			fmt.Println("Server shutting down.")
			end <- true;
			break;
		}

		if err == io.EOF {
			end <- true;
			break;
		}
	}
}

func processPacket(conn *net.UDPConn, buf []byte) {
	message := protocol.Decode(buf)

	//See if we already have a session with this id open.
	sesschan, ok := chanMapGet(message.Sessionid)
	if !ok {
		//make sure the message is HELLO
		if (message.Command == protocol.HELLO) {
			sesschan = make(chan protocol.P0Pmessage, 10)
			chanmap[message.Sessionid] = sesschan
			sessionWaitGroup.Add(1)
			go func() {
				defer sessionWaitGroup.Done()
				handleClient(conn)
			}()
		} else {
			//Something is broken here or with the client, quit.
			end <- true
		}
	}

	sesschan<-message;
}

func handleClient(conn *net.UDPConn) {
	fmt.Println("Session created")

	/*sessionstate := 0
	var seqnum uint32
	seqnum = 0;
	var sessionid uint32*/

	for {
		select {
		case _,ok := <- quit:
			if !ok {
				break;
			}
		}

		/*//If the magic field does not match, silently discard the packet
		if (message.Magic != protocol.MAGIC) {
			continue
		}

		//Check that version number is correct
		if (message.Version != protocol.VERSION) {
			continue
		}

		//Check if we lost packets
		if (message.Sequencenumber != seqnum) {
			//Lost message.Sequencenumber - seqnum packets.
		}

		seqnum = message.Sequencenumber

		//TODO: set session id if first message
		if (message.Sessionid != sessionid){
			//TODO: do something
		}

		if (sessionstate == protocol.HELLO) { //Ready to say hello.
			if (message.Command == protocol.HELLO) {
				//Correct, send hello message back
			} else {
				//Error
			}
		} else if sessionstate == protocol.ALIVE { //Ready to receive

		} else { //Session state is GOODBYE and session should have already ended
			//TODO: throw error
		}*/
	}

	//TODO: send goodbye message
}

func chanMapGet(sessionId uint32) (chan protocol.P0Pmessage, ok bool){
	chanmaplock.Lock()
	defer chanmaplock.Unlock()
	sesschan, ok := chanmap[sessionId]
	if !ok {
		return nil, false
	} else {
		return sesschan
	}
}

func chanMapDelete(sessionId uint32) {
	chanmaplock.Lock()
	delete(chanmap, sessionId)
	chanmaplock.Unlock()
}