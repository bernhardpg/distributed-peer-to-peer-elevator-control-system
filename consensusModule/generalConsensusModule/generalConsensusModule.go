package generalConsensusModule

import (
	"../../network"
)

type Req struct {
	State ReqState;
	AckBy []network.NodeID
}

type ReqState int
const (
	Inactive ReqState = iota
	PendingAck
	Confirmed
	Unknown
)

func UniqueIDSlice(IDSlice []network.NodeID) []network.NodeID {

    keys := make(map[network.NodeID]bool)
    list := []network.NodeID{} 

    for _, entry := range IDSlice {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}

func containsElement(s [] network.NodeID, e network.NodeID) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

//Returns true if primaryList contains listFraction
func containsList(primaryList [] network.NodeID, listFraction [] network.NodeID) bool {
    for _, a := range listFraction {
        if !containsElement(primaryList, a){
        	return false
        }
    }
    return true
}

func Merge (pLocal *Req, remote Req, localID network.NodeID, peersList [] network.NodeID)(bool){
	newConfirmedOrInactiveFlag := false

	switch (*pLocal).State {

	case Inactive:
		if remote.State == PendingAck {
			*pLocal = Req {
				State: PendingAck, 
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
			}
		}

	case PendingAck:
		(*pLocal).AckBy = UniqueIDSlice(append(remote.AckBy, localID))

		if (remote.State == Confirmed) || containsList((*pLocal).AckBy, peersList) {
			(*pLocal).State = Confirmed
			newConfirmedOrInactiveFlag = true
			//Signaliser confirmed
		}
		

	case Confirmed:
		(*pLocal).AckBy = UniqueIDSlice(append(remote.AckBy, localID))

		if remote.State == Inactive {
			*pLocal = Req {
				State: Inactive,
				AckBy: nil,
			}
			newConfirmedOrInactiveFlag = true

		}


	case Unknown:
		switch remote.State {


		case Inactive:
			*pLocal = Req {
				State: Inactive,
				AckBy: nil,
			}
			newConfirmedOrInactiveFlag = true


		case PendingAck:
			*pLocal = Req {
				State: PendingAck,
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
			}


		case Confirmed:
			*pLocal = Req {
				State: Confirmed,
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
				//Signaliser confirmed
			}
			newConfirmedOrInactiveFlag = true

		}
	}
	
	return newConfirmedOrInactiveFlag
}
