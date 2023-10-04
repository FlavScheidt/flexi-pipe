package main

import (  
  	"fmt"
  	"time"
  	"log"
    // "os/exec"

    // influxdb2 "github.com/influxdata/influxdb-client-go/v2"
    "golang.org/x/crypto/ssh"
)



func runNode(hostname string, config *ssh.ClientConfig, duration time.Duration) {

    log.Println("Enter function")
    results := make(chan string, 10)
    timeout := time.After(duration)

    cmd := "cd "+PATH+" && "+GOPATH+"go run . -machine=node\n"
    log.Println(cmd)
    
    go func(hostname string) {
        results <- executeCmd(cmd, hostname, config)
    }(hostname)
    log.Println(hostname+":Command sent")
    
    // executeCmd(cmd, hostname, config)

    select {
        case res := <-results:
            fmt.Print(res)
        case <-timeout:
            log.Println(hostname, ": Influx Load Timed out!")
            return
    }
}
