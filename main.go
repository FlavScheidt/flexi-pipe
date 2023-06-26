package main

import (
    // "bytes"
    // "log"
    "io"
    "io/ioutil"
    "os"
    "time"
    "log"
    "flag"
    "strings"
    "os/exec"


	"golang.org/x/crypto/ssh"
	kh "golang.org/x/crypto/ssh/knownhosts"
	
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
 )


// -----------------------------------------
//      Set paths
// -----------------------------------------
var PATH="/root/flexi-pipe/"
var TOOLS_PATH=PATH+"rippledTools/"
var RIPPLED_PATH="/opt/local/bin/"
var RIPPLED_CONFIG="/root/config/rippled.cfg"
var RIPPLED_QUORUM="15"

var GOSSIPSUB_PATH="/root/gossipGoSnt/"
var GOPATH="/usr/local/go/bin/"
var GOSSIPSUB_PARAMETERS=PATH+"config/parameters.csv"

var NODES_CONFIG=PATH+"rippledTools/ConfigCluster/ClusterConfig.csv"

var PUPPET="liberty"

// var experiment="unl"

func main() {
	//------------------------------------------
	//	Proccess flags
	//------------------------------------------
	// Important note: we are actually reading the parameters for gs from a file
	// However, we need to also be able to read them from command line
	//because we will use the commandline to start the puppet
	machineFlag := flag.String("machine", "master", "Is this machine a master or a puppet? Deafult is master")
  	experimentType := flag.String("type", "unl", "Type of experiment. Default is unl")

  	runtime := flag.Duration("runtime", 900*time.Second, "Time for each test, counting from the start of gossipsub. Default is 900s (15 min)")
  	// runtime := flag.Duration("runtime", 100*time.Second, "Time for each test, counting from the start of gossipsub. Default is 900s (15 min)")


    d := flag.String("d", "8", "")
    dlo := flag.String("dlo", "6", "")
    dhi := flag.String("dhi", "12", "")
    dscore := flag.String("dscore", "4", "")
    dlazy := flag.String("dlazy", "8", "")
    dout := flag.String("dout", "2", "")
    gossipFactor := flag.String("gossipFactor", "0.25", "")

    // InitialDelay := flag.Duration("InitialDelay", 100 * time.Millisecond, "")
    // Interval := flag.Duration("Interval", 1 * time.Second, "")

    flag.Parse()

	machine := strings.ToLower(*machineFlag)
	topology := strings.ToLower(*experimentType)
	runTime := *runtime

    // -----------------------------------------
    //      Set log file
    //			Just the go logging feature, nothing special
    // -----------------------------------------
    currentTime := time.Now()
    LOG_FILE := PATH+"log_"+currentTime.Format("01022006_15_04_05")+"_"+topology+".out"
    // open log file
    logFile, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        log.Panic(err)
    }
    defer logFile.Close()

    mw := io.MultiWriter(os.Stdout, logFile)
    log.SetOutput(mw)
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)

    // -----------------------------------------
    //		Nodes
    // -----------------------------------------
    //Read nodes name from config file
    hosts, error := readNodesFile(NODES_CONFIG)
    if err != nil {
        log.Panic(error)
    }

    log.Printf("%+v\n", hosts)

    // -----------------------------------------
    //		SSH config
    // -----------------------------------------
    user := "root"
    timeout := 4800 * time.Second

	// key, err := ioutil.ReadFile("/root/.ssh/id_rsa")
	key, err := ioutil.ReadFile("/root/.ssh/id_rsa")
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	// hostKeyCallback, err := kh.New("/root/.ssh/known_hosts")
	hostKeyCallback, err := kh.New("/root/.ssh/known_hosts")
	if err != nil {
		log.Fatal("could not create hostkeycallback function: ", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	if machine == "master" {
		// -----------------------------------------
	    // 		Parameters for GossipSub
	    // -----------------------------------------
	    // Read nodes name from config file
	    paramsList, er := readParamsFile(GOSSIPSUB_PARAMETERS)
	    if er != nil {
	        log.Panic(error)
	    }

	    log.Printf("%+v\n", paramsList)

	    // -----------------------------------------
	    // 		Generate config for the chosen topology
	    // -----------------------------------------
	    cmd := exec.Command("/bin/bash", TOOLS_PATH+"ConfigCluster/generate_config_rippled.sh "+topology)
	    log.Println(TOOLS_PATH+"ConfigCluster/generate_config_rippled.sh "+topology)
		stdout, err := cmd.Output()
		if err != nil {
		    log.Println(err.Error())
		}
		// Print the output
		log.Println("Generating rippled config: "+string(stdout))

	    // -----------------------------------------
	    // 		Start rippled
	    // -----------------------------------------
	    start := []string{
	    		"nohup " + RIPPLED_PATH+"rippled --conf="+RIPPLED_CONFIG+" --silent --net --quorum "+RIPPLED_QUORUM+" & \n",
	    		"disown -h %1\n",
	    	}
   		stop := RIPPLED_PATH+"rippled --conf="+RIPPLED_CONFIG+" stop & \n"

   		// experiment.start = time.Now()

		for _, param := range paramsList {
			// -----------------------------------------
		    // 		Clean logs and rippled databases
		    // -----------------------------------------
		    cmd = exec.Command("/bin/bash", TOOLS_PATH+"NewRun/prepareNewRun.sh")
		    log.Println(TOOLS_PATH+"NewRun/prepareNewRun.sh")
			stdout, err = cmd.Output()
			if err != nil {
			    log.Println(err.Error())
			}
			// Print the output
			log.Println("Cleaning logs and databases: "+string(stdout))

		    for _, hostname := range hosts {
		    	log.Println(hostname+" Starting rippled")
			    go remoteShell(start, hostname, config)
		    }
		    time.Sleep(60 * time.Second)

		    //Connect to puppet server and start GossipSub
		    log.Println("Connecting to ", PUPPET)
    		go runPuppet(topology, config, timeout, param)

    		//Start rippled monitor
    		go rippledMonitor(hosts, config, runTime)

    		time.Sleep(runTime)

    		//Stop rippled
    		for _, hostname := range hosts {
		    	log.Println(hostname+" Stoping rippled")
			    go executeCmd(stop, hostname, config)
		    }
		    time.Sleep(100)
		}
	} else if machine == "node" {

		// Create write client
	    writeClient := influxdb2.NewClient(url, token)

		//Get hostname
		hostname, err := os.Hostname()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}

		//Load the traces
		loadTraces(hostname, writeClient)

	} else if machine == "puppet" {

		//Get parameters from command line
		param := OverlayParams{
	        d:            *d,
	        dlo:          *dlo,
	        dhi:          *dhi,
	        dscore:       *dscore,
	        dlazy:        *dlazy,
	        dout:         *dout,
	        gossipFactor: *gossipFactor,
	    }

	    // Create struct with experiment info for the database
	    experiment := Experiment{
	    	topology:		topology,
	    	runtime:		uint64(runTime),
	    	overlayParams:	param,
	    	start:			time.Now(),
	    }

		//Connect and start gossipsub
		gossipsub := "cd "+GOSSIPSUB_PATH+" && "+GOPATH+"go run . -type="+topology+" -d="+param.d+" -dlo="+param.dlo+" -dhi="+param.dhi+" -dscore="+param.dscore+" -dlazy="+param.dlazy+" -dout="+param.dout+"\n"
		for _, hostname := range hosts {
			log.Println("Starting GossipSub")
			go executeCmd(gossipsub, hostname, config)
		}

		time.Sleep(runTime)

		//Kill all execution
		kill := "pkill -9 gossipGoSnt && pkill -9 rippled\n"
		for _, hostname := range hosts {
			log.Println("Stoping GossipSub")
			go executeCmd(kill, hostname, config)
		}

		experiment.end = time.Now()
		time.Sleep(30)

	    // Create write client
	    writeClient := influxdb2.NewClient(url, token)

	    //Load experiment data into influxdb
		pt := pointData{
			timestamp : experiment.start,
			measurement: "experiment",
			tags: map[string]string{
				"topology": experiment.topology,
			},
			fields: map[string]interface{}{
	 				"endTime": 		experiment.end,
				 	"runtime": 		experiment.runtime,
				 	"d": 			experiment.overlayParams.d,
				 	"dlo":          experiment.overlayParams.dlo,
			        "dhi":          experiment.overlayParams.dhi,
			        "dscore":       experiment.overlayParams.dscore,
			        "dlazy":        experiment.overlayParams.dlazy,
			        "dout":         experiment.overlayParams.dout,
			        "gossipFactor": experiment.overlayParams.gossipFactor,
			},
		}
		writeDB(pt, writeClient)
		log.Println("point created")

	    // -----------------------------------------
	    // 		Load traces into db
	    // -----------------------------------------
	    log.Println("Starting nodes to load traces into influx")
	    for _, hostname := range hosts {
	    	go runNode(hostname, config, timeout)
	    }
	    // time.Sleep(100 * time.Second)
	}
	// time.Sleep(100 * time.Second)
	select {}
 }