package gotcpd

import (
    "os"
    "fmt"
    "net"
    "sync"
    "time"
    "bufio"
    "strings"
    "strconv"
    "testing"
)

const serverAddr = "127.0.0.1:7777"

// Global variable initialized in main() and asserted in TestPassingArbitraryUserDataPtr()
var myDataPtr string

// Helper function
func testPtr(conn net.Conn, t *testing.T) {
    fmt.Fprintf(conn, "GET ptr\n")
    message, _ := bufio.NewReader(conn).ReadString('\n')
    if message != myDataPtr {
        t.Errorf("%#v != %#v\n", message, myDataPtr)
    }
}

// Helper function
func testData(conn net.Conn, t *testing.T) {
    fmt.Fprintf(conn, "GET data\n")
    message, _ := bufio.NewReader(conn).ReadString('\n')
    if message != "my data\n" {
        t.Errorf("%#v != 'my data'\n", message)
    }
}

// Check if server responds correctly
func TestEcho(t *testing.T) {
    conn, _ := net.Dial("tcp", serverAddr)
    fmt.Fprintf(conn, "GET echo\n")
    message, _ := bufio.NewReader(conn).ReadString('\n')
    if message != "echo\n" {
        t.Errorf("%#v != 'echo'\n", message)
    }
    conn.Close()
}

// Check if arbitrary user data is correctly passed by pointer
func TestPassingArbitraryUserDataPtr(t *testing.T) {
    // Multiple requests on single connection
    conn, _ := net.Dial("tcp", serverAddr)
    testPtr(conn, t)
    testPtr(conn, t)
    testPtr(conn, t)
    conn.Close()

    // New connection
    conn1, _ := net.Dial("tcp", serverAddr)
    testPtr(conn1, t)
    conn1.Close()

    // New connection
    conn2, _ := net.Dial("tcp", serverAddr)
    testPtr(conn2, t)
    conn2.Close()
}

// Check if arbitrary user data is correctly passed
func TestPassingArbitraryUserData(t *testing.T) {
    // Multiple requests on single connection
    conn, _ := net.Dial("tcp", serverAddr)
    testData(conn, t)
    testData(conn, t)
    testData(conn, t)
    conn.Close()

    // New connection
    conn1, _ := net.Dial("tcp", serverAddr)
    testData(conn1, t)
    conn1.Close()

    // New connection
    conn2, _ := net.Dial("tcp", serverAddr)
    testData(conn2, t)
    conn2.Close()
}

// Test long key
func TestLongKey(t *testing.T) {
    // Create a string 32MB long
    var key string = "x"
    for i := 0; i<25; i++ {
        key = key + key
    }

    conn, _ := net.Dial("tcp", serverAddr)
    fmt.Fprintf(conn, "GET %s\n", key)
    message, _ := bufio.NewReader(conn).ReadString('\n')
    if len(message) != len(key) + len("\n") {
        t.Errorf("Length of key %d != %d\n", len(message), len(key) + len("\n"))
    }
    conn.Close()
}

// Test multiple requests sent on one connection.
// Responses should come in order and processing time must indicate concurrent processing.
func TestResponsesOrder(t *testing.T) {
    conn, _ := net.Dial("tcp", serverAddr)
    reader := bufio.NewReader(conn)

    start := time.Now()

    fmt.Fprintf(conn, "GET sleep 2\nGET sleep 0\nGET sleep 3\nGET sleep 1\n")

    if message, _ := reader.ReadString('\n'); message != "sleep 2\n" {
        t.Errorf("%#v != 'sleep 2'\n", message)
    }

    if message, _ := reader.ReadString('\n'); message != "sleep 0\n" {
        t.Errorf("%#v != 'sleep 0'\n", message)
    }

    if message, _ := reader.ReadString('\n'); message != "sleep 3\n" {
        t.Errorf("%#v != 'sleep 3'\n", message)
    }

    if message, _ := reader.ReadString('\n'); message != "sleep 1\n" {
        t.Errorf("%#v != 'sleep 1'\n", message)
    }

    end := time.Now()
    elapsed := end.Sub(start)

    if elapsed.Milliseconds() > 3100 {
        t.Errorf("Execution time %v > 3 seconds \n", elapsed)
    }
}

// Run N requests: 1 request per connection
func TestConcurrency1(t *testing.T) {
    const N = 100
    var waitgroup sync.WaitGroup
    waitgroup.Add(N)

    start := time.Now()

    for i := 0; i<N; i++ {
        go func() {
            conn, _ := net.Dial("tcp", serverAddr)
            fmt.Fprintf(conn, "GET sleep 1\n")
            bufio.NewReader(conn).ReadString('\n')
            waitgroup.Done()
        }()
    }
    waitgroup.Wait()

    end := time.Now()
    elapsed := end.Sub(start)

    if elapsed.Milliseconds() > 1100 {
        t.Errorf("Execution time %v > 1 second \n", elapsed)
    }
}

// Run N requests on one connection
func TestConcurrency2(t *testing.T) {
    const N = 100
    var waitgroup sync.WaitGroup
    waitgroup.Add(N)

    conn, _ := net.Dial("tcp", serverAddr)
    reader := bufio.NewReader(conn)
    start := time.Now()

    for i := 0; i<N; i++ {
        go func() {
            fmt.Fprintf(conn, "GET sleep 1\n")
            waitgroup.Done()
        }()
    }
    waitgroup.Wait()

    for i := 0; i<N; i++ {
        reader.ReadString('\n')
    }

    end := time.Now()
    elapsed := end.Sub(start)

    if elapsed.Milliseconds() > 1100 {
        t.Errorf("Execution time %v > 1 second \n", elapsed)
    }
}

// Test different delimiter
func TestLongDelimiter(t *testing.T) {
    conn, _ := net.Dial("tcp", "127.0.0.1:7778")
    fmt.Fprintf(conn, "GET echo\r\r.\r\r")
    message, _ := bufio.NewReader(conn).ReadString('\n')
    if message != "My proto is CFCF.CFCF\n" {
        t.Errorf("%#v != 'My proto is CFCF.CFCF'\n", message)
    }
    conn.Close()
}

type UserData struct {
    mydata string
}

func worker(request string, userdata interface{}) string {
    data := userdata.(*UserData)

    if request == "GET ptr\n" {
        return fmt.Sprintf("%p\n", &data.mydata)
    }

    if request == "GET data\n" {
        return fmt.Sprintf("%s\n", data.mydata)
    }

    // Split string by spaces
    fields := strings.Fields(request)

    // GET sleep 2
    if fields[1] == "sleep" {
        seconds, _ := strconv.Atoi(fields[2])
        time.Sleep(time.Duration(seconds) * time.Second)
    }

    return fmt.Sprintf("%s\n", strings.Join(fields[1:], " "))
}

func worker2(request string, userdata interface{}) string {
    return fmt.Sprintf("My proto is CFCF.CFCF\n")
}

func TestMain(m *testing.M) {

    // Run main server
    userdata := UserData{"my data"}
    myDataPtr = fmt.Sprintf("%p\n", &userdata.mydata)
    go RunServer(serverAddr, "\n", worker, &userdata)

    // Run second server
    go RunServer("127.0.0.1:7778", "\r\r.\r\r", worker2, nil)

    // Wait a bit to have tcp accepting loops ready
    time.Sleep(100 * time.Millisecond)

    // Run tests
    code := m.Run()
    os.Exit(code)
}
