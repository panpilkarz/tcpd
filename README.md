Framework for tcp daemons handling arbitrary request->response text protocols.

### Minimal example
```
package main

import (
    "gotcpd"
)

func callback(request string, userdata interface{}) string {
    // do you stuff and return response
    return "OK\n"
}

func main() {
    // Accept tcp connections on 127.0.0.1:7777
    // "\n\n" indicates the end of request.
    // Pass the request to callback().
    gotcpd.RunServer("127.0.0.1:7777", "\n\n", callback, nil)
} 
```

### Example with arbitrary data passed to callback (use for init and shared data)
```
package main

import (
    "gotcpd"
)

type SharedData struct {
    // your data
}

func callback(request string, sharedData interface{}) string {
    data := shareData.(*SharedData)
    // do you stuff and return response
    return "OK\n"
}

func main() {
    // Accept tcp connections on 127.0.0.1:7777
    // "\n\n" indicates the end of request.
    // Pass the request and shared data to callback().
    mydata := SharedData
    gotcpd.RunServer("127.0.0.1:7777", "\n\n", callback, &mydata)
} 
```

### Test
```
go test -v
```

