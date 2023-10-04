package main

import (
    "time"
    "log"
    // "math/rand"


	"golang.org/x/crypto/ssh"
	// kh "golang.org/x/crypto/ssh/knownhosts"
)

func runTransactions(hostname string, runtime time.Duration, config *ssh.ClientConfig) {
    log.Println("Start transactions")

    time.Sleep(2 * time.Minute)

    duration := runtime - (180*time.Second)
    cmd := "cd "+PATH+" && python3 transactions.py"

    for start := time.Now(); time.Since(start) < duration; {
        executeCmd(cmd, hostname, config)
        log.Println("Trasaction completed")
        time.Sleep(1*time.Second)
    }
}