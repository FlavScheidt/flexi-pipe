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

var TRACES_PATH=PATH+"/traces/"

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

  	// runtime := flag.Duration("runtime", 900*time.Second, "Time for each test, counting from the start of gossipsub. Default is 900s (15 min)")
  	runtime := flag.Duration("runtime", 100*time.Second, "Time for each test, counting from the start of gossipsub. Default is 900s (15 min)")


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
    LOG_FILE := "./log_"+currentTime.Format("01022006_15_04_05")+"_"+topology+".out"
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
	    // 		Clean logs and rippled databases
	    // -----------------------------------------
	   	cd := "cd "+TOOLS_PATH+"/NewRun/"
	   	clean := "./prepareNewRun.sh"

	    cmd := exec.Command(cd, clean)
		stdout, err := cmd.Output()
		if err != nil {
		    log.Println(err.Error())
		}
		// Print the output
		log.Println("Cleaning logs and databases: "+string(stdout))

	    // -----------------------------------------
	    // 		Generate config for the chosen topology
	    // -----------------------------------------

	    // -----------------------------------------
	    // 		Start rippled
	    // -----------------------------------------
	    start := []string{
	    		"nohup " + RIPPLED_PATH+"rippled --conf="+RIPPLED_CONFIG+" --silent --net --quorum "+RIPPLED_QUORUM+" & \n",
	    		"disown -h %1\n",
	    	}
   		stop := RIPPLED_PATH+"rippled --conf="+RIPPLED_CONFIG+" stop & \n"

		for _, param := range paramsList {
		    for _, hostname := range hosts {
		    	log.Println(hostname+" Starting rippled")
			    go remoteShell(start, hostname, config)
		    }
		    time.Sleep(60 * time.Second)

		    //Connect to puppet server and start GossipSub
		    log.Println("Connecting to ", PUPPET)
    		go runPuppet(topology, config, timeout, param)

    		//Start rippled monitor
    		// go rippledMonitor(hosts, config, runTime)

    		time.Sleep(runTime)

    		//Stop rippled
    		for _, hostname := range hosts {
		    	log.Println(hostname+" Stoping rippled")
			    go executeCmd(stop, hostname, config)
		    }
		}

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

		kill := "pkill -9 gossipGoSnt && pkill -9 rippled\n"
		for _, hostname := range hosts {
			log.Println("Stoping GossipSub")
			go executeCmd(kill, hostname, config)
		}

		experiment.end = time.Now()

	    // Create write client
	    writeClient := influxdb2.NewClient(url, token)
	    // Define write API
	    writeAPI := writeClient.WriteAPI(org, bucket)

	    // Get errors channel
		errorsCh := writeAPI.Errors()
		// Create go proc for reading and logging errors
		go func() {
			for err := range errorsCh {
				log.Printf("write error: %s\n", err.Error())
			}
		}()

		//Timestamp is the begning of the execution
		// timestamp := time.Unix(0, int64(experiment.timestamp))
		// point := influxdb2.NewPoint(
		// 	"experiment",
		// 	map[string]string{
		// 	"topology": experiment.topology,
		// 	},
	 // 		map[string]interface{}{
	 // 				"endTime": 		experiment.end,
		// 		 	"runtime": 		experiment.runtime,
		// 		 	"d": 			experiment.overlayParams.d,
		// 		 	"dlo":          experiment.overlayParams.dlo,
		// 	        "dhi":          experiment.overlayParams.dhi,
		// 	        "dscore":       experiment.overlayParams.dscore,
		// 	        "dlazy":        experiment.overlayParams.dlazy,
		// 	        "dout":         experiment.overlayParams.dout,
		// 	        "gossipFactor": experiment.overlayParams.gossipFactor,
		// 		 },
		// 	timestamp)
		// writeAPI.WritePoint(point)


		pt := pointData{
			timestamp : experiment.start,
			measurement: "message",
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
		log.Println("point created")

		writeDB(pt, writeClient)

		// timestamp := time.Unix(0, int64(experiment.timestamp))
		// point := influxdb2.NewPoint(
		// 	"experiment",
		// 	map[string]string{
		// 	"topology": experiment.topology,
		// 	},
	 // 		map[string]interface{}{
	 // 				"endTime": 		experiment.end,
		// 		 	"runtime": 		experiment.runtime,
		// 		 	"d": 			experiment.overlayParams.d,
		// 		 	"dlo":          experiment.overlayParams.dlo,
		// 	        "dhi":          experiment.overlayParams.dhi,
		// 	        "dscore":       experiment.overlayParams.dscore,
		// 	        "dlazy":        experiment.overlayParams.dlazy,
		// 	        "dout":         experiment.overlayParams.dout,
		// 	        "gossipFactor": experiment.overlayParams.gossipFactor,
		// 		 },
		// 	timestamp)
		// writeAPI.WritePoint(point)

	    // -----------------------------------------
	    // 		Load traces into db
	    // -----------------------------------------
	    for _, hostname := range hosts {
	    	//copy traces
	    	copy := "scp "+hostname+":"+GOSSIPSUB_PATH+"/trace.json "+TRACES_PATH+"/trace_"+hostname+".json"

	    	cmd := exec.Command(copy)
		    stdout, err := cmd.Output()
		    if err != nil {
		        log.Println(err.Error())
		        return
		    }
		    // Print the output
		    log.Println("Copying trace from "+hostname+": "+string(stdout))


		    //load traces
		    // go loadTraces(hostname, writeClient)
	    }


	}
	// time.Sleep(100 * time.Second)
	select {}
 }