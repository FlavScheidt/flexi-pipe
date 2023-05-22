package main

import (  

	// "context"
  	"fmt"
  	// "os"
  	"time"
  	"log"
  	"io"
  	// "reflect"
  	// "encoding/json"
  	// "go-ndjson"

  	"github.com/bfontaine/jsons"
  	// "github.com/influxdata/influxdb-client-go/v2/api/write"
  	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var org = "XRPL Project"
var bucket = "test"
var url = "https://eu-central-1-1.aws.cloud2.influxdata.com"
var token = "-rwAjn0_D1heOSIfReDrjPbnR7m_wgAg_O_RWvcnZ7qYI-jngsa-jlhk1qw2BlCullTfRuZurAqRQywV6klR_g=="

func writeTopic(measurement string, data map[string]interface{}, timestamp time.Time, writeClient influxdb2.Client) () {
    writeAPI := writeClient.WriteAPI(org, bucket)

	j := data[measurement]
	join := j.(map[string]interface{})

	point := influxdb2.NewPoint(
			measurement,
			map[string]string{
				"peerID": data["peerID"].(string),
			},
			 map[string]interface{}{
			 	"topic": join["topic"].(string),
			 },
			timestamp)
		writeAPI.WritePoint(point)
		// writeAPI.Flush()
}

func writePublishMessage(measurement string, data map[string]interface{}, timestamp time.Time, writeClient influxdb2.Client) {
	writeAPI := writeClient.WriteAPI(org, bucket)

	j := data[measurement]
	join := j.(map[string]interface{})

	point := influxdb2.NewPoint(
			measurement,
			map[string]string{
				"peerID": data["peerID"].(string),
			},
			 map[string]interface{}{
			 	"topic": join["topic"].(string),
			 	"messageID": join["messageID"].(string),
			 },
			timestamp)
		writeAPI.WritePoint(point)
		// writeAPI.Flush()
}
func writeMessage(measurement string, data map[string]interface{}, timestamp time.Time, writeClient influxdb2.Client) () {
    writeAPI := writeClient.WriteAPI(org, bucket)

	j := data[measurement]
	join := j.(map[string]interface{})

	point := influxdb2.NewPoint(
			measurement,
			map[string]string{
				"peerID": data["peerID"].(string),
			},
 			map[string]interface{}{
				 	"topic": join["topic"].(string),
				 	"messageID": join["messageID"].(string),
				 	"receivedFrom": join["receivedFrom"].(string),
				 },
			timestamp)
		writeAPI.WritePoint(point)
		// writeAPI.Flush()
}

func writeRPC(measurement string, data map[string]interface{}, timestamp time.Time, writeClient influxdb2.Client) {
	writeAPI := writeClient.WriteAPI(org, bucket)

	//GeneralPoint
	j := data[measurement]
	join := j.(map[string]interface{})
	// writeAPI.Flush()

	meta := join["meta"].(map[string]interface{})
	//Read meta
	if _, ok := meta["messages"]; ok {
		m := meta["messages"]
		mt := m.([]interface{})

		point := influxdb2.NewPoint(
			measurement,
			map[string]string{
			"peerID": data["peerID"].(string),
			},
 			map[string]interface{}{
				 	"receivedFrom": join["receivedFrom"].(string),
				 	"type": "messages",
				 },
			timestamp)
		writeAPI.WritePoint(point)

		for _, mess := range mt {
			message := mess.(map[string]interface{})
			point = influxdb2.NewPoint(
				"RPCmessages",
				map[string]string{
					"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 "topic": message["topic"].(string),
					 "messageID": message["messageID"].(string),
					},
				timestamp)
		}
		writeAPI.WritePoint(point)
		// writeAPI.Flush()
	} else if _, ok := meta["control"]; ok {
		c := meta["control"]
		control := c.(map[string]interface{})

		if _, ok := control["ihave"]; ok {
			ihave := control["ihave"].([]interface{})

			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
	 			map[string]interface{}{
					 	"receivedFrom": join["receivedFrom"].(string),
					 	"type": "ihave",
					 },
				timestamp)
			writeAPI.WritePoint(point)
			
			for _, mess := range ihave {
				message := mess.(map[string]interface{})
				ids := message["messageIDs"].([]interface{})

				nMessages := len(ids)
				// log.Println(nMessages)

				point := influxdb2.NewPoint(
					"ihave",
					map[string]string{
						"peerID": data["peerID"].(string),
					},
		 			map[string]interface{}{
						 	"topic": message["topic"].(string),
						 	"messages": nMessages,
						 },
					timestamp)
				writeAPI.WritePoint(point)
				// writeAPI.Flush()
			}
		} else if _, ok := control["graft"]; ok {
			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 	"receivedFrom": join["receivedFrom"].(string),
					 	"type": "graft",
					 },
				timestamp)
			writeAPI.WritePoint(point)
		} else if _, ok := control["prune"]; ok {
			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 	"receivedFrom": join["receivedFrom"].(string),
					 	"type": "prune",
					 },
				timestamp)
			writeAPI.WritePoint(point)
		} else {
			log.Println("Format not recognized:")
			log.Printf("%+v\n", data)
		}
	} else if _, ok := meta["subscription"]; ok {
		point := influxdb2.NewPoint(
			measurement,
			map[string]string{
			"peerID": data["peerID"].(string),
			},
	 		map[string]interface{}{
				 	"receivedFrom": join["receivedFrom"].(string),
				 	"type": "subscription",
				 },
			timestamp)
		writeAPI.WritePoint(point)
	} else {
		log.Println("Format not recognized:")
		log.Printf("%+v\n", data)
	}
}

func writeSentRPC(measurement string, data map[string]interface{}, timestamp time.Time, writeClient influxdb2.Client) {
	writeAPI := writeClient.WriteAPI(org, bucket)

	//GeneralPoint
	j := data[measurement]
	join := j.(map[string]interface{})
	// writeAPI.Flush()

	meta := join["meta"].(map[string]interface{})
	//Read meta
	if _, ok := meta["messages"]; ok {
		m := meta["messages"]
		mt := m.([]interface{})

		point := influxdb2.NewPoint(
			measurement,
			map[string]string{
			"peerID": data["peerID"].(string),
			},
 			map[string]interface{}{
				 	"sendTo": join["sendTo"].(string),
				 	"type": "messages",
				 },
			timestamp)
		writeAPI.WritePoint(point)

		for _, mess := range mt {
			message := mess.(map[string]interface{})
			point = influxdb2.NewPoint(
				"RPCmessages",
				map[string]string{
					"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 "topic": message["topic"].(string),
					 "messageID": message["messageID"].(string),
					},
				timestamp)
		}
		writeAPI.WritePoint(point)
		// writeAPI.Flush()
	} else if _, ok := meta["control"]; ok {
		c := meta["control"]
		control := c.(map[string]interface{})

		if _, ok := control["iwant"]; ok {
			iwant := control["iwant"].([]interface{})

			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
	 			map[string]interface{}{
					 	"sendTo": join["sendTo"].(string),
					 	"type": "iwant",
					 },
				timestamp)
			writeAPI.WritePoint(point)
			
			for _, mess := range iwant {
				message := mess.(map[string]interface{})
				ids := message["messageIDs"].([]interface{})

				nMessages := len(ids)
				// log.Println(nMessages)

				point := influxdb2.NewPoint(
					"iwant",
					map[string]string{
						"peerID": data["peerID"].(string),
					},
		 			map[string]interface{}{
						 	"messages": nMessages,
						 },
					timestamp)
				writeAPI.WritePoint(point)
				// writeAPI.Flush()
			}
		} else if _, ok := control["ihave"]; ok {
			ihave := control["ihave"].([]interface{})

			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
	 			map[string]interface{}{
					 	"sendTo": join["sendTo"].(string),
					 	"type": "ihave",
					 },
				timestamp)
			writeAPI.WritePoint(point)
			
			for _, mess := range ihave {
				message := mess.(map[string]interface{})
				ids := message["messageIDs"].([]interface{})

				nMessages := len(ids)
				// log.Println(nMessages)

				point := influxdb2.NewPoint(
					"ihave",
					map[string]string{
						"peerID": data["peerID"].(string),
					},
		 			map[string]interface{}{
						 	"topic": message["topic"].(string),
						 	"messages": nMessages,
						 },
					timestamp)
				writeAPI.WritePoint(point)
				// writeAPI.Flush()
			}	
		} else if _, ok := control["graft"]; ok {
			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 	"sendTo": join["sendTo"].(string),
					 	"type": "graft",
					 },
				timestamp)
			writeAPI.WritePoint(point)
		} else if _, ok := control["prune"]; ok {
			point := influxdb2.NewPoint(
				measurement,
				map[string]string{
				"peerID": data["peerID"].(string),
				},
		 		map[string]interface{}{
					 	"sendTo": join["sendTo"].(string),
					 	"type": "prune",
					 },
				timestamp)
			writeAPI.WritePoint(point)
		} else {
			log.Println("Format not recognized:")
			log.Printf("%+v\n", data)
		}
	} else if _, ok := meta["subscription"]; ok {
		point := influxdb2.NewPoint(
			measurement,
			map[string]string{
			"peerID": data["peerID"].(string),
			},
	 		map[string]interface{}{
				 	"sendTo": join["sendTo"].(string),
				 	"type": "subscription",
				 },
			timestamp)
		writeAPI.WritePoint(point)
	} else {
		log.Println("Format not recognized:")
		log.Printf("%+v\n", data)
	}
}

func loadTraces() {
    // Create write client
    writeClient := influxdb2.NewClient(url, token)

    // Define write API
    writeAPI := writeClient.WriteAPI(org, bucket)

    // Get errors channel
	errorsCh := writeAPI.Errors()
	// Create go proc for reading and logging errors
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
		}
	}()

    //Load json
	fileName := "/home/flav/gossip/flexi-pipe/trace.json"
	bytes := jsons.NewFileReader(fileName)

	//Read json
    if err := bytes.Open(); err != nil {
        log.Fatal(err)
    }
    defer bytes.Close()

    //Loop over json objects
    for {
        var data map[string]interface{}
        if err := bytes.Next(&data); err != nil {
            if err == io.EOF {
                break
            }
            log.Fatal(err)
        }

        // log.Printf("got %+v", data)

        //TimeStamp
        tm := data["timestamp"].(float64)
        timestamp := time.Unix(0, int64(tm))

       	// Write data of the general message
		point := influxdb2.NewPoint(
			"message",
			map[string]string{
				"peerID": data["peerID"].(string),
			},
			 map[string]interface{}{
			 	// "size": size,
			 	"type": data["type"],
			 },
			timestamp)

		writeAPI.WritePoint(point)
		// writeAPI.Flush()

		//Specific metrics
		switch tp := data["type"].(float64); tp {
		case 0:
			writePublishMessage("publishMessage", data, timestamp, writeClient)
		case 11:
			writeTopic("graft", data, timestamp, writeClient)
		case 9:
			log.Println("Message type join")
		case 4:
			log.Println("Message type addPeer")
		case 5:
			log.Println("Message type removePeer")
		case 12:
			writeTopic("prune", data, timestamp, writeClient)
		case 3:
			writeMessage("deliverMessage", data, timestamp, writeClient)
		case 2:
			writeMessage("duplicateMessage", data, timestamp, writeClient)
		case 6:
			writeRPC("recvRPC", data, timestamp, writeClient)
		case 7:
			writeSentRPC("sendRPC", data, timestamp, writeClient)
		default:
			log.Println("Format not recognized")
			log.Printf("got %+v", data)
		}
	}
	writeAPI.Flush()
}

