package main

import(
	"github.com/mdaigle/SimpleServer/protocol"
	"fmt"
	"net"
	"os"
	"time"
	"bufio"
	"io"
	"sync"
)

// Global mapping from session ids to channels for session threads
var chanmap = make(map[uint32](chan protocol.P0Pmessage))
var chanmaplock sync.Mutex

// Global counter for number of responses sent
var server_seq_num uint32 = 0
var server_seq_num_lock sync.Mutex

// A wait group incremented for each client thread opened.
// Decremented when client threads die.
var sessionWaitGroup sync.WaitGroup

// When closed, signals that threads should end execution
var quit = make(chan bool)
// A write to this channel signals that the server should shut down
var end = make(chan bool)

func main() {
	portnum := os.Args[1];
	udpaddr, err := net.ResolveUDPAddr("udp4", ":"+portnum)
	if err != nil {
		fmt.Println("error resolving address")
	}

	go broadcastQuit()
	go readIn()

	fmt.Printf("Waiting on port %v...\n", portnum)
	conn, err := net.ListenUDP("udp4", udpaddr)

	for {
		//Check if we should quit
		cont := true
		select{
		case _,ok := <-quit:
			if !ok {
				cont = false
			}
		default:

		}

		if (!cont) {break}

		//fmt.Println("About to read from udp")
		buf := make([]byte, 512)
		conn.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
		_, addr, err := conn.ReadFromUDP(buf[0:])

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				//fmt.Println("Read timed out")
				continue
			}
			fmt.Println(err.Error())
			end <- true
			break
		}

		processPacket(conn, addr, buf)
	}
	sessionWaitGroup.Wait()
	conn.Close()
}

// Closes quit when a shutdown command is received through the end channel.
func broadcastQuit() {
	<-end
	close(quit)
}

// Waits for input on stdin. Sends shutdown command if 'q' or EOF detected.
func readIn() {
	//fmt.Println("In input routine.")
	reader := bufio.NewReader(os.Stdin)

	for {
		text, err := reader.ReadString('\n')

		if text == "q\n" {
			//post to channel or global
			end <- true
			return
		}

		if err == io.EOF {
			end <- true
			return
		}
	}
}

func processPacket(conn *net.UDPConn, addr *net.UDPAddr, buf []byte) {
	message := protocol.Decode(buf)

	//TODO: maybe move validation to separate function

	// If the magic value is corrupt, silently discard the message
	if message.Magic != protocol.MAGIC {
		return
	}

	// If the message is using an incorrect version of the protocol, discard it
	if message.Version != protocol.VERSION {
		return
	}

	// See if we already have a session with this id open.
	sesschan, ok := chanMapGet(message.Sessionid)
	if !ok {
		//make sure the message is HELLO
		if (message.Command == protocol.HELLO) {
			//TODO: try an unbuffered channel and let this routine block until the session routine is ready
			// will that cause ordering issues?
			sesschan = make(chan protocol.P0Pmessage, 10)
			chanmap[message.Sessionid] = sesschan
			go handleClient(conn, addr, sesschan, message)
			return
		} else {
			// Something is broken here or with the client, quit.
			end <- true
		}
	}

	sesschan<-message
}

func handleClient(conn *net.UDPConn, addr *net.UDPAddr, sesschan chan protocol.P0Pmessage, initMessage protocol.P0Pmessage) {
	var client_seq_num uint32 = 0;
	if initMessage.Sequencenumber != client_seq_num {
		fmt.Println("ERROR: Non-zero initial sequence number:", initMessage.Sequencenumber)
		//definitely close session
	}

	var sessionid uint32 = initMessage.Sessionid
	fmt.Printf("%v [%v] Session created\n", sessionid, initMessage.Sequencenumber)

	//Respond with a HELLO message
	var hello_response protocol.P0Pmessage
	hello_response.Magic = protocol.MAGIC
	hello_response.Version = protocol.VERSION
	hello_response.Command = protocol.HELLO
	hello_response.Sequencenumber = getServerSeqNum()
	hello_response.Sessionid = sessionid

	hello_buf := protocol.Encode(hello_response)

	num_written, err := conn.WriteToUDP(hello_buf, addr)
	if (num_written == 0 || err != nil) {
		fmt.Println("ERROR: Write failed")
	}

	sessTimer := time.NewTimer(5 * time.Second)

	for {
		cont := true
		select {
		case <- sessTimer.C:
			cont = false
			break
		case _, ok := <-quit:
			if !ok {
				cont = false
				break
			}
		case message, _ := <- sesschan:
			//Check for packet ordering issues


			if (message.Sequencenumber < client_seq_num) {
				cont = false
				break
			}
			if (client_seq_num != 0 && message.Sequencenumber == client_seq_num) {
				fmt.Printf("%v [%v] Duplicate packet\n", sessionid, message.Sequencenumber)
				break
			}
			if (message.Sequencenumber > client_seq_num + 1) {
				//We lost packets
				for i := client_seq_num + 1; i < message.Sequencenumber; i++ {
					fmt.Printf("%v [%v] Lost packet\n", sessionid, i)
				}
			}
			client_seq_num = message.Sequencenumber

			if (message.Command == protocol.GOODBYE) {
				fmt.Printf("%v [%v] GOODBYE from client.\n", sessionid, message.Sequencenumber);
				cont = false
				break
			}

			if (message.Command != protocol.DATA) {
				// incorrect command
				cont = false
				break
			}

			// Print data to standard out
			fmt.Printf("%v [%v] " + string(message.Data[:]) + "\n", sessionid, message.Sequencenumber)

			//Respond with an ALIVE message
			var response protocol.P0Pmessage
			response.Magic = protocol.MAGIC
			response.Version = protocol.VERSION
			response.Command = protocol.ALIVE
			response.Sequencenumber = getServerSeqNum()
			response.Sessionid = sessionid

			buf := protocol.Encode(response)

			num_written, err := conn.WriteToUDP(buf, addr)
			if (num_written == 0 || err != nil) {
				fmt.Println("ERROR: Write failed")
				//close session?
			}
			sessTimer.Reset(5 * time.Second)
		default:
			//Include default case so that we are non-blocking and can check up on the timer
		}
		if (!cont) {break}
	}

	var response protocol.P0Pmessage
	response.Magic = protocol.MAGIC
	response.Version = protocol.VERSION
	response.Command = protocol.GOODBYE
	response.Sequencenumber = getServerSeqNum()
	response.Sessionid = sessionid

	buf := protocol.Encode(response)

	num_written, err = conn.WriteToUDP(buf, addr)
	if (num_written == 0 || err != nil) {
		fmt.Println("ERROR: Write failed")
	}

	chanMapDelete(sessionid)
	close(sesschan)
}

func chanMapGet(sessionId uint32) (chan protocol.P0Pmessage, bool){
	chanmaplock.Lock()
	defer chanmaplock.Unlock()
	sesschan, ok := chanmap[sessionId]
	if !ok {
		return nil, false
	} else {
		return sesschan, true
	}
}

func chanMapDelete(sessionId uint32) {
	chanmaplock.Lock()
	delete(chanmap, sessionId)
	chanmaplock.Unlock()
}

func getServerSeqNum() (uint32) {
	server_seq_num_lock.Lock()
	defer server_seq_num_lock.Unlock()
	num := server_seq_num
	server_seq_num += 1
	return num
}