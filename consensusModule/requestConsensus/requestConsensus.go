package requestConsensus

import (
	"../../fsm"
)


type Req struct {
	state ReqState;
	ackBy []fsm.NodeID
}

type ReqState int
const (
	Inactive ReqState = iota
	PendingAck
	Confirmed
	Unknown
)

func uniqueIDSlice(IDSlice []fsm.NodeID) []fsm.NodeID {
    keys := make(map[fsm.NodeID]bool)
    list := []fsm.NodeID{} 
    for _, entry := range IDSlice {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}

func containsElement(s [] fsm.NodeID, e fsm.NodeID) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

//Returns true if primaryList contains listFraction
func containsList(primaryList [] fsm.NodeID, listFraction [] fsm.NodeID) bool {
    for _, a := range listFraction {
        if !containsElement(primaryList, a){
        	return false
        }
    }
    return true
}

func merge (pLocal *Req, pRemote *Req, localID fsm.NodeID, peersList [] fsm.NodeID){

	switch (*pLocal).state {

	case Inactive:
		if (*pRemote).state == PendingAck {
			*pLocal = Req {
				state: PendingAck, 
				ackBy: uniqueIDSlice(append((*pRemote).ackBy, localID)),
			}
		}

	case PendingAck:
		(*pLocal).ackBy = uniqueIDSlice(append((*pRemote).ackBy, localID))

		if (*pRemote).state == Confirmed || containsList((*pLocal).ackBy, peersList) {
			(*pLocal).state = Confirmed
			//Signaliser confirmed
		}
		

	case Confirmed:
		(*pLocal).ackBy = uniqueIDSlice(append((*pRemote).ackBy, localID))

		if (*pRemote).state == Inactive {
			*pLocal = Req {
				state: Inactive,
				ackBy: nil,
			}
		}


	case Unknown:
		switch (*pRemote).state {


		case Inactive:
			*pLocal = Req {
				state: Inactive,
				ackBy: nil,
			}


		case PendingAck:
			*pLocal = Req {
				state: PendingAck,
				ackBy: uniqueIDSlice(append((*pRemote).ackBy, localID)),
			}


		case Confirmed:
			*pLocal = Req {
				state: Confirmed,
				ackBy: uniqueIDSlice(append((*pRemote).ackBy, localID)),
				//Signaliser confirmed
			}	
		}
	}
}