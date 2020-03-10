package gotcpd

import (
    "net"
    "log"
    "bufio"
    "strings"
    "encoding/binary"
)

// Function provided by the user.
// Arbitrary user data provided to RunServer() is available by pointer in @userdata.
// Must return a string to be send to tcp client.
type HandlerFunc func(request string, userdata interface{}) string

type Response struct {
    reqNum int
    value string
    finished bool
}

// Waits for responses from the connection requests workers
// Receive responses via @queue channel
// Send to @finished channel that all responses are flushed
func responder(writer *bufio.Writer, queue <-chan Response, finished chan<- bool) {
    var waitingForReqNum = 0
    var finishedAtReqNum = -1

    // Store responses in the map.
    // @key: reqNum
    // @value: response plain text
    responses := make(map[int]string)

    //log.Printf("Start responder\n")
    for msg := range queue {
        responses[msg.reqNum] = msg.value

        if finishedAtReqNum == -1 && msg.finished {
            finishedAtReqNum = msg.reqNum
        }

        // Try to flush responses queue
        for {
            if value, ok := responses[waitingForReqNum]; ok {
                // Dont write to client which has finished
                if finishedAtReqNum == -1 {
                    writer.WriteString(value)
                    writer.Flush()
                }
                //log.Printf("[%d] Response: %#v\n", waitingForReqNum, value)
                delete(responses, waitingForReqNum)
                waitingForReqNum += 1
                continue
            }
            break
        }

        if finishedAtReqNum >= 0 && waitingForReqNum > finishedAtReqNum {
            finished <- true
        }
    }

    //log.Printf("Responder has finished\n")
}

func cb(request string, queue chan<- Response, callback HandlerFunc, userdata interface{}, reqNum int, finished bool) {

    if finished {
        // Communicate to the responder that client has finished
        queue <- Response{reqNum, "", true}
        return
    }

    // Execute user-defined function
    value := callback(request, userdata)

    // Send the response to the responder
    queue <- Response{reqNum, value, false}
}

func handleConnection(conn net.Conn, requestDelimiter string, callback HandlerFunc, userdata interface{}) {
    //log.Printf("[%v] Got new connection\n", conn)

    var request string

    var delimiter byte = 0
    var textSize int32 = 0

    if requestDelimiter != "" {
        delimiter = byte(requestDelimiter[len(requestDelimiter)-1])
    }

    var reqNum = 0
    var queue = make(chan Response)
    var finished = make(chan bool)

    reader := bufio.NewReader(conn)
    writer := bufio.NewWriter(conn)

    defer conn.Close()
    defer close(queue)
    defer close(finished)

    // Start responder goroutine that writes to the client
    go responder(writer, queue, finished);

    for {
        // Handle '[n][text]' protocol where @n is binary int32 @text size
        if delimiter == 0 {

            if textSize == 0 {
                // Read [n] part of the protcol
                err := binary.Read(reader, binary.LittleEndian, &textSize)
                if err != nil {
                    //log.Printf("Client left... %v\n", err)
                    go cb("", queue, callback, userdata, reqNum, true)
                    break
                }
            }

            buf := make([]byte, 32768)
            n, err := reader.Read(buf)

            if err != nil {
                //log.Printf("Client left... %v\n", err)
                go cb("", queue, callback, userdata, reqNum, true)
                break
            }

            request += string(buf[0:n])

            // Read of [text] part of protocol is complete
            if int32(len(request)) == textSize {
                go cb(request, queue, callback, userdata, reqNum, false)
                textSize = 0
                reqNum += 1
                request = ""
            }

        } else {
            // Handle delimiter based text protocol

            s, err := reader.ReadString(delimiter)

            if err != nil {
                //log.Printf("Client left... %v\n", err)
                go cb("", queue, callback, userdata, reqNum, true)
                break
            }

            request += s

            if strings.HasSuffix(request, requestDelimiter) {
                //log.Printf("[%v] Request: %#v\n", reqNum, request)
                go cb(request, queue, callback, userdata, reqNum, false)
                reqNum += 1
                request = ""
            }
        }
    }

    // Wait for responder goroutine to finish
    <- finished
}

func RunServer(addr string, requestDelimiter string, callback HandlerFunc, userdata interface{}) {
    log.Printf("Binding to tcp:%v\n", addr)

    listener, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatal("tcp server listener error:", err)
    }

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Fatal("tcp server accept error", err)
        }

        go handleConnection(conn, requestDelimiter, callback, userdata)
    }
}
