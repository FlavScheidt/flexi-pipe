package main

import (  
  	"fmt"
  	"time"
  	"log"
    "os/exec"

    influxdb2 "github.com/influxdata/influxdb-client-go/v2"
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

func loadLogs(hostname string,  writeClient influxdb2.Client) {
    log.Println("Enter loadLogs")

    //Format file for loading
    cmd := "cat "+RIPPLED_LOG+"debug.log | grep 'LedgerConsensus' | grep 'Built ledger' | cut -d ' ' -f1,2,7 | sed 's/.$//' |  sed 's/#//' | sed 's/ /./' | sed 's/ /,/' > log_temp.out"
    
    _, err := exec.Command("bash","-c",cmd).Output()
    if err != nil {
        log.Printf("Failed to execute command: %s", cmd)
        return
    }

    // Define write API
    writeAPI := writeClient.WriteAPI(org, bucket)
    log.Println("Write api created")

    // Get errors channel
    errorsCh := writeAPI.Errors()
    // Create go proc for reading and logging errors
    go func() {
        for err := range errorsCh {
            log.Printf("write error: %s\n", err.Error())
        }
    }()

    //Read data and load into the db
    records, err := readData("./log_temp.out")
    if err != nil {
        log.Fatal(err)
        return
    }

    for _, record := range records {

        timestamp, err := time.Parse("2006-Jan-01.15:04:05.000000000", record[0])
        log.Println("Datetime:", timestamp.String())
        if err != nil {
            fmt.Println(err)
        }

        // Write data of the general message
        point := influxdb2.NewPoint(
            "consensus",
            map[string]string{
                "node": hostname,
            },
            map[string]interface{}{
                // "size": size,
                "ledger": record[1],
            },
            timestamp)

        writeAPI.WritePoint(point)
        writeAPI.Flush()
    }

    //Remove temp file
    cmd = "rm -rf log_temp.out"
    
    _, err = exec.Command("bash","-c",cmd).Output()
    if err != nil {
        log.Printf("Failed to execute command: %s", cmd)
        return
    }
}
