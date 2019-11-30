package main

import (
    "gotcpd"
)

func worker(request string, userdata interface{}) string {
    return "HTTP/1.1 200 OK\r\nConnection: Keep-Alive\r\nKeep-Alive: timeout=60, max=1000000\r\nContent-Length: 5\r\n\r\nHello"
}

func main() {
    gotcpd.RunServer("127.0.0.1:8000", "\r\n\r\n", worker, nil)
}
