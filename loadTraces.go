package main

import (  

	// "context"
  	"fmt"
  	// "os"
  	"time"
  	"log"
  	"io"
  	// "encoding/json"
  	// "go-ndjson"

  	"github.com/bfontaine/jsons"
  	// "github.com/influxdata/influxdb-client-go/v2/api/write"
  	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)



func loadTraces() {
    // Create write client
    url := "https://eu-central-1-1.aws.cloud2.influxdata.com"
    token := "-rwAjn0_D1heOSIfReDrjPbnR7m_wgAg_O_RWvcnZ7qYI-jngsa-jlhk1qw2BlCullTfRuZurAqRQywV6klR_g=="
    writeClient := influxdb2.NewClient(url, token)

    // Define write API
    org := "XRPL Project"
    bucket := "genericTest"
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
			 	"type": data["type"],
			 },
			timestamp)

		writeAPI.WritePoint(point)
		log.Printf("got %+v", data)

		//Specific metrics
		if data["type"].(float64) == 9 {
			j := data["join"]
			join := j.(map[string]interface{})

			point := influxdb2.NewPoint(
			"join",
			map[string]string{
				"peerID": data["peerID"].(string),
			},
			 map[string]interface{}{
			 	"topic": join["topic"].(string),
			 },
			timestamp)
			writeAPI.WritePoint(point)
		}



		// if err := writeAPI.WritePoint(context.Background(), point); err != nil {
		//    return fmt.Errorf("write API write point: %s", err)
		// }

		// time.Sleep(1 * time.Second) // separate points by 1 second
	}
	writeAPI.Flush()
}


// func traceJoin(data map[string]interface{}, writeAPI influxdb2.WriteAPI, timestamp time.Time) {
// 	point := influxdb2.NewPoint(
// 			"join",
// 			map[string]string{
// 				"peerID": data["peerID"].(string),
// 			},
// 			 map[string]interface{}{
// 			 	"topic": data["join"],
// 			 },
// 			timestamp)
// 	writeAPI.WritePoint(point)
// }

        // if data["type"] == 9 {

        // }

	// //Unmarshal and store in an interface (we don't know wich type of message we are dealing yet)
	// var result interface{}
	// err = json.Unmarshal(bytes, &result)
	// if err != nil {
	// 	panic(err)
	// }

	// data := result.(map[string]interface{})

	// if data["type"].(float64) == 9 {
	// 	fmt.Println("type 9")
	// }

	// Write data
 //    for key := range data {
 //      point := influxdb2.NewPointWithMeasurement("bytes").
 //        AddTag("peerID", data[key]["peerID"].(string)).
 //        AddTag("type", data[key]["type"].(string))

	// writeAPI.WritePoint(context.Background(), point)
 //      // if err := writeAPI.WritePoint(context.Background(), point); err != nil {
 //      //   return fmt.Errorf("write API write point: %s", err)
 //      // }

 //      time.Sleep(1 * time.Second) // separate points by 1 second
 //    }


// 	//Load generic message data
// 	timestamp, err := time.Parse(
//             "2006 02 Jan 03:04 PM -0700", 
//             bytes["timestamp"].(string))
// 	if err != nil {
// 		panic(err)
// 	}

// 	data := Message{}
// 	data.tp =  	bytes["type"].(uint16)
// 	data.peerId = bytes["peerID"].(string)

// 	p := influxdb2.NewPoint(
// 			"bytes",
// 			map[string]string{
// 				"peerID": data.peerId,
// 			},
// 			 map[uint16]interface{ 
// 			 	"type": data.tp,
// 			},
// 			timestamp)

// 	client := influxdb2.NewClient(url, token)
// 	writer := client.WriteAPI(org, bucket)
// 	writer.WritePoint(p)
// 	client.Close()

// }


// func unMarshalUnstructured(fileName string)
// {

// 	bytes, err := os.ReadFile(fileName)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var result interface{}
// 	err = json.Unmarshal(bytes, &result)
// 	if err != nil {
// 		panic(err)
// 	}

// 	bytes := result.(map[string]interface{})

// 	// for k, v := range bytes {
// 	//     switch vv := v.(type) {
// 	// 	    case string:
// 	// 	        fmt.Println(k, "is string", vv)
// 	// 	    case float64:
// 	// 	        fmt.Println(k, "is float64", vv)
// 	// 	    case []interface{}:
// 	// 	        fmt.Println(k, "is an array:")
// 	// 	        for i, u := range vv {
// 	// 	            fmt.Println(i, u)
// 	// 	        }
// 	// 	    default:
// 	// 	        fmt.Println(k, "is of a type I don't know how to handle")
// 	// 	}
//  //    }
// }



// 	// The object stored in the "birds" key is also stored as 
// 	// a map[string]any type, and its type is asserted from
// 	// the `any` type
// 	msg := result["type"].(map[string]any)

// 	for key, value := range birds {
// 	  // Each value is an `any` type, that is type asserted as a string
// 	  fmt.Println(key, value.(string))
// }
