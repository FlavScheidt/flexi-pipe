package main

import (
    "time"
    // "log"
    // "math/rand"


	"golang.org/x/crypto/ssh"
	// kh "golang.org/x/crypto/ssh/knownhosts"
)

func runTransactions(hostname string, runtime time.Duration, config *ssh.ClientConfig) {
    time.Sleep(2 * time.Minute)

    duration := runtime - (140*time.Second)
    cmd := "cd "+PATH+" && python3 transactions.py"

    for start := time.Now(); time.Since(start) < duration; {
        go executeCmd(cmd, hostname, config)

        time.Sleep(1*time.Second)
    }
}