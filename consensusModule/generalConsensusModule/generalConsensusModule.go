package requestConsensus

import (
	"../../fsm"
	"../../elevio"
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

func merge (pLocal *Req, remote Req, localID fsm.NodeID, peersList [] fsm.NodeID)(bool){
	newConfirmedOrInactiveFlag := false

	switch (*pLocal).state {

	case Inactive:
		if remote.state == PendingAck {
			*pLocal = Req {
				state: PendingAck, 
				ackBy: uniqueIDSlice(append(remote.ackBy, localID)),
			}
		}

	case PendingAck:
		(*pLocal).ackBy = uniqueIDSlice(append(remote.ackBy, localID))

		if (remote.state == Confirmed) || containsList((*pLocal).ackBy, peersList) {
			(*pLocal).state = Confirmed
			newConfirmedOrInactiveFlag = true
			//Signaliser confirmed
		}
		

	case Confirmed:
		(*pLocal).ackBy = uniqueIDSlice(append(remote.ackBy, localID))

		if remote.state == Inactive {
			*pLocal = Req {
				state: Inactive,
				ackBy: nil,
			}
			newConfirmedOrInactiveFlag = true

		}


	case Unknown:
		switch remote.state {


		case Inactive:
			*pLocal = Req {
				state: Inactive,
				ackBy: nil,
			}
			newConfirmedOrInactiveFlag = true


		case PendingAck:
			*pLocal = Req {
				state: PendingAck,
				ackBy: uniqueIDSlice(append(remote.ackBy, localID)),
			}


		case Confirmed:
			*pLocal = Req {
				state: Confirmed,
				ackBy: uniqueIDSlice(append(remote.ackBy, localID)),
				//Signaliser confirmed
			}
			newConfirmedOrInactiveFlag = true

		}
	}
	
	return newConfirmedOrInactiveFlag
}
