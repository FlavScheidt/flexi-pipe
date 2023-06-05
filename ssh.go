package main

import (
    "bytes"
    // "log"
    // "io"
    // "io/ioutil"
    // "time"
    "log"

    scp "github.com/bramvdbogaerde/go-scp"
    "github.com/bramvdbogaerde/go-scp/auth"
    "os"
    "context"

	"golang.org/x/crypto/ssh"
	// kh "golang.org/x/crypto/ssh/knownhosts"
 )

func scpTrace(hostname string) {
    clientConfig, _ := auth.PrivateKey("root", "/root/.ssh/id_rsa", ssh.InsecureIgnoreHostKey())

    // For other authentication methods see ssh.ClientConfig and ssh.AuthMethod

    // Create a new SCP client
    client := scp.NewClient(hostname+":22", &clientConfig)

    // Connect to the remote server
    err := client.Connect()
    if err != nil {
        log.Println("Couldn't establish a connection to the remote server ", err)
        return
    }

    // Open a file
    f, _ := os.Open(TRACES_PATH+"traces_"+hostname+".json")

    // Close client connection after the file has been copied
    defer client.Close()

    // Close the file after it has been copied
    defer f.Close()

    // Finaly, copy the file over
    // Usage: CopyFromFile(context, file, remotePath, permission)

    // the context can be adjusted to provide time-outs or inherit from other contexts if this is embedded in a larger application.
    err = client.CopyFromFile(context.Background(), *f, PATH+"trace.json", "0655")

    if err != nil {
        log.Println("Error while copying file ", err)
    }
    log.Println("Success")
    client.Close()
    f.Close()
}


func executeCmd(cmd string, hostname string, config *ssh.ClientConfig) string {//, client *ssh.Client) string {
    
    var stdoutBuf bytes.Buffer

    client, err := ssh.Dial("tcp", hostname+":22", config)
    if err != nil {
        log.Fatalf("unable to connect: %v", err)
    }
    defer client.Close()
    
    ss, err := client.NewSession()
    if err != nil {
        log.Fatal("unable to create SSH session: ", err)
    }
    defer ss.Close()
    ss.Setenv("GOPATH", GOPATH)
    
    // Creating the buffer which will hold the remotly executed command's output.
    ss.Stdout = &stdoutBuf
    ss.Stdout = os.Stdout
    ss.Stderr = os.Stderr
    ss.Run(cmd)

    // Let's print out the result of command.
    log.Println(stdoutBuf.String())

    return hostname + ": " + stdoutBuf.String()
}

func remoteShell(commands []string, hostname string, config *ssh.ClientConfig,) {

    client, err := ssh.Dial("tcp", hostname+":22", config)
        if err != nil {
            log.Fatalf("unable to connect: %v", err)
        }
    defer client.Close()

    // Create sesssion
    sess, err := client.NewSession()
    if err != nil {
        log.Fatal("Failed to create session: ", err)
    }
    defer sess.Close()

    sess.Setenv("GOPATH", GOPATH)

    // StdinPipe for commands
    stdinBuf, err := sess.StdinPipe()
    if err != nil {
        log.Fatal(err)
    }

    // Uncomment to store output in variable
    var b bytes.Buffer
    sess.Stdout = &b
    sess.Stderr = &b

    // Enable system stdout
    // Comment these if you uncomment to store in variable
    // sess.Stdout = os.Stdout
    // sess.Stderr = os.Stderr

    // Start remote shell
    err = sess.Shell()
    if err != nil {
        log.Fatal(err)
    }

    // log.Println(hostname, ": Running command | ", cmd)
    // stdinBuf.Write([]byte(cmd))
    // time.Sleep(20 * time.Second)

    // disown := "disown -h %1\n"
    // log.Println(hostname, ": disown")
    // stdinBuf.Write([]byte(disown))
    // send the commands
    // commands := []string{
    //     "pwd",
    //     "whoami",
    //     "echo 'bye'",
    //     "exit",
    // }

    for _, cmd := range commands {
        // _, err = log.Fprintf(stdin, "%s\n", cmd)
        _, err = stdinBuf.Write([]byte(cmd))
        if err != nil {
            log.Fatal(err)
        }
    }

    //Wait for sess to finish
    err = sess.Wait()
    if err != nil {
        log.Fatal(err)
    }
    // Uncomment to store in variable
    // log.Println(hostname, ": ", b.String())

}

