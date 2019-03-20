package hallConsensusModule

import(
	"../elevio"
	"../stateHandler"
	"fmt"
	"fsm"
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


func setLocalHallOrder(localID fsm.NodeID, buttonPress elevio.ButtonEvent, locallyAssignedHallOrders [][]Req) {
	fmt.Println("Setting order in local order matrix!")

	locallyAssignedHallOrders[buttonPress.Floor][buttonPress.Button] = Req {
		state: PendingAck,
		ackBy: []fsm.NodeID{localID},
	}
}

	// TODO set lights
	// TODO send orders to fsm!


func clearOrdersAtFloor(localID fsm.NodeID, floor int, locallyAssignedHallOrders [][] Req, confirmedHallOrders [][] bool) {
	inactiveReq := Req {
		state: Inactive, 
		ackBy: []fsm.NodeID{localID},
	}
	locallyAssignedHallOrders[floor] = [] Req {inactiveReq, inactiveReq}
}

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
 

func OrderConsensus(localID fsm.NodeID,
	numFloors int, 
	CompletedOrderChan <-chan int, 
	NewOrderChan <-chan elevio.ButtonEvent,
	PeersListChan <-chan [] fsm.NodeID,
	RemoteHallOrdersChan <-chan [][] Req,
	ConfirmedHallOrdersToIOChan chan<- [][] bool,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	LocalHallOrdersToNewtorkChan chan<- [][] Req) {

	var locallyAssignedHallOrders = make([][] Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)
	peersList := [] fsm.NodeID{}

// Initialize all to unknown
	for floor := range locallyAssignedHallOrders {
		for orderReq := range locallyAssignedHallOrders[floor] {

			locallyAssignedHallOrders[floor][orderReq] = Req {
				state: Unknown,
				ackBy: nil,
			}
			
			confirmedHallOrders[floor] = [] bool{false, false}
		}
	}
	fmt.Println("hallConsensusModule initialized")

	for {

		select{

		case a := <- NewOrderChan:
			setLocalHallOrder(localID, a, locallyAssignedHallOrders)
			//Update network	

		case a := <- CompletedOrderChan:
			clearOrdersAtFloor(localID, a, locallyAssignedHallOrders, confirmedHallOrders)
			//Update optimal assigner
			//Update IO
			//Update network
		
		case a := <- PeersListChan:
			peersList = a
		

		case a := <- RemoteHallOrdersChan:
			remoteHallOrders := a

			for floor := range locallyAssignedHallOrders {
				for orderReq := range locallyAssignedHallOrders[floor]{

					//Switching on local state
					switch locallyAssignedHallOrders[floor][orderReq].state {

					case Inactive:
						if (remoteHallOrders[floor][orderReq].state == PendingAck){
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: PendingAck, 
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
							}
						}

					case PendingAck:
						if (remoteHallOrders[floor][orderReq].state == Confirmed){
							locallyAssignedHallOrders[floor][orderReq].state = Confirmed
						}
						locallyAssignedHallOrders[floor][orderReq].ackBy = uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID))
						


					case Confirmed:
						if (remoteHallOrders[floor][orderReq].state == Inactive){
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Inactive,
								ackBy: nil,
							}
						}else {
							locallyAssignedHallOrders[floor][orderReq].ackBy = uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID))
						}


					case Unknown:
						switch remoteHallOrders[floor][orderReq].state {

						case Inactive:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Inactive,
								ackBy: nil,
							}


						case PendingAck:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: PendingAck,
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
							}


						case Confirmed:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Confirmed,
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
							}



						}

					}
				}

				
			}
		}

	}

}