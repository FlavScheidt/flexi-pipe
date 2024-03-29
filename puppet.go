package main

import (
    // "bytes"
    "fmt"
    // "io"
    // "io/ioutil"
    // "os"
    "time"
    "log"
    // "os/exec"

    // influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"golang.org/x/crypto/ssh"
	// kh "golang.org/x/crypto/ssh/knownhosts"
)

func runPuppet(experiment string, config *ssh.ClientConfig, duration time.Duration, param OverlayParams){ //, runTime time.Duration) {

    results := make(chan string, 10)
    timeout := time.After(duration)

    cmd := "cd "+PATH+" && "+GOPATH+"go run . -type="+experiment+" -machine=puppet -parameter="+param.parameter+
            " -d="+param.d+
            " -dlo="+param.dlo+
            " -dhi="+param.dhi+
            " -dscore="+param.dscore+
            " -dlazy="+param.dlazy+
            " -dout="+param.dout+
            " -gossipFactor="+param.gossipFactor+
            " -initialDelay="+param.initialDelay+
            " -interval="+param.interval+
            "\n"
    hostname := PUPPET
    
    go func(hostname string) {
        results <- executeCmd(cmd, hostname, config)
    }(hostname)
    
    // executeCmd(cmd, hostname, config)

    select {
        case res := <-results:
            fmt.Print(res)
        case <-timeout:
            log.Println(hostname, ": Puppet Timed out!")
            return
    }
}


