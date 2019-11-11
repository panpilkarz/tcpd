package gotcpd

import (
    "net"
    "log"
    "bufio"
    "strings"
)

// Function provided by the user.
// Arbitrary user data provided to RunServer() is available by pointer in @userdata.
// Must return a string to be send to tcp client.
type HandlerFunc func(request string, userdata interface{}) string

type Response struct {
    req_id int
    value string
}

// Waits for responses from the connection requests workers
func responder(writer *bufio.Writer, queue <-chan Response) {
    var waiting_for_req_id = 0

    // Store responses in the map.
    // @key: req_id
    // @value: response text
    responses := make(map[int]string)

    //log.Printf("Start responder\n")
    for response := range queue {
        responses[response.req_id] = response.value

        // Try to flush responses queue
        for {
            value, ok := responses[waiting_for_req_id]
            if ok {
                writer.WriteString(value)
                writer.Flush()
                //log.Printf("[%d] Response: %#v\n", waiting_for_req_id, value)
                delete(responses, waiting_for_req_id)
                waiting_for_req_id += 1
                continue
            }
            break
        }
    }

    log.Printf("Responder has finished\n")
}


func callbackWrapper(req_id int, request string, queue chan<- Response, callback HandlerFunc, userdata interface{}) {
    // Execute user-defined function
    value := callback(request, userdata)

    // Send the response to the responder
    queue <- Response{req_id, value}
}

func handleConnection(conn net.Conn, requestDelimiter string, callback HandlerFunc, userdata interface{}) {
    var err error = nil
    var request string
    var s string
    var delimiter byte = byte(requestDelimiter[len(requestDelimiter)-1])

    reader := bufio.NewReader(conn)
    writer := bufio.NewWriter(conn)

    var req_id = 0
    var queue = make(chan Response)

    go responder(writer, queue);

    //log.Printf("[%v] Got new connection\n", conn)

    defer conn.Close()
    defer close(queue)

    for {
        s, err = reader.ReadString(delimiter)

        if err != nil {
            //log.Println("client left...")
            break
        }

        request += s

        if strings.HasSuffix(request, requestDelimiter) {
            //log.Printf("[%v] Got new request: %#v\n", req_id, request)
            go callbackWrapper(req_id, request, queue, callback, userdata)
            req_id += 1
            request = ""
        }
    }
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
