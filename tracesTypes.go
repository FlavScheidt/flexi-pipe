package main

import (  
)


type Message struct {
	TimeStamp 	string  `json:"timestamp"`
	peerId		string	`json:"peerID"`
	tp			uint16	`json:"type"`
}


//type = 4
//"addPeer":{"peerID":"ACQIARIgutFeCGDvY8CEGLKJLbwhzAojBcANvaPtrqW25pK5kS0=","proto":"/meshsub/1.1.0"}
type AddPeer struct {
	peerId		string	`json:"peerID"`
	proto		string 	`json:"proto"`
}

//type = 5
//"removePeer":{"peerID":"ACQIARIgy9UsLGQXZgxfmle06NvWnMG6X8Lgry2zmzr66ebUJUM="}
type RemovePeer struct {
	peerId		string 	`json:"peerID"`
}

//type = 9
//"join":{"topic":"validator:orange"}
type Join struct {
	topic		string	`json:"topic"`
}

//type = 10
// type Leave {

// }

//type = 11
//"graft":{"peerID":"ACQIARIgFcrOE6zggqC78TeZqH1n+PmFRXIUXDO+uNuTnLCc7fg=","topic":"validator:orange"}}
type Graft struct {
	peerId		string 	`json:"peerID"`
	topic		string	`json:"topic"`	
}

//type = 12
//"prune":{"peerID":"ACQIARIgBeo1PFEVY0FfdOnl91TUUgiSekNH9/+KNZKwGVrBeKk=","topic":"validator:blue"}}
type Prune struct {
	peerId		string 	`json:"peerID"`
	topic		string	`json:"topic"`	
}

// type ValidateMessage {

// }

//type = 3
//"deliverMessage":{"messageID":"ACQIARIgD0L9DFUrEMrmskLqnXInFpXWSpAe/CjcDO6NgA8ZVBIXWc67G6okag==","topic":"validator:blue","receivedFrom":"ACQIARIgD0L9DFUrEMrmskLqnXInFpXWSpAe/CjcDO6NgA8ZVBI="
type DeliverMessage struct {
	messageId		string	`json:"messadeID"`
	topic			string	`json:"topic"`
	receivedFrom	string	`json:receivedFrom"`
}

//type = 1
// type RejectMessage {

// }

//type = 2
//"duplicateMessage":{"messageID":"ACQIARIgD0L9DFUrEMrmskLqnXInFpXWSpAe/CjcDO6NgA8ZVBIXWc67G6okaQ==","receivedFrom":"ACQIARIg48RRwjv88DzyjXOEIFC0zdxIAsXkF9DSfh2nGSkkaj0=","topic":"validator:orange"}}
type DuplicateMessage struct {
	messageId		string	`json:"messadeID"`
	topic			string	`json:"topic"`
	receivedFrom	string	`json:receivedFrom"`
}

// type ThrottlePeer {

// }

//type = 6
//{"type":6,"peerID":"ACQIARIgo+G6XPg1Kjdwpc5M7CxrGlQXg5se6bHlFaYJjYRDX9Y=","timestamp":1682603247903609362,
//	"recvRPC":{"receivedFrom":"ACQIARIg9J81Yj/smCNF6/9o0XZmWCIQLH5scTrDUQ/5FydA3/g=",
//				"meta":{"subscription":[{"subscribe":true,"topic":"validator:cyano"},{"subscribe":true,"topic":"validator:fucsia"},{"subscribe":true,"topic":"validator:green"},{"subscribe":true,"topic":"validator:orange"},{"subscribe":true,"topic":"validator:pink"},{"subscribe":true,"topic":"validator:red"},{"subscribe":true,"topic":"validator:yellow"},{"subscribe":true,"topic":"validator:blue"}]}}}
// type RecvRPC {

// }

// //type = 7
// type SendRPC  {

// }

// //type = 8
// type DropRPC {

// }

// type UndeliverableMessage {

// }