package hallConsensusModule

import(
	"../elevio"
	"../stateHandler"
	"fmt"
	"fsm"
	"./requestConsensus"
	)




func setLocalHallOrder(localID fsm.NodeID, buttonPress elevio.ButtonEvent, locallyAssignedHallOrders [][]requestConsensus.Req) {
	fmt.Println("Setting order in local order matrix!")

	locallyAssignedHallOrders[buttonPress.Floor][buttonPress.Button] = requestConsensus.Req {
		state: PendingAck,
		ackBy: []fsm.NodeID{localID},
	}
}

	// TODO set lights
	// TODO send orders to fsm!


func clearOrdersAtFloor(localID fsm.NodeID, floor int, locallyAssignedHallOrders [][] requestConsensus.Req, confirmedHallOrders [][] bool) {
	inactiveReq := requestConsensus.Req {
		state: Inactive, 
		ackBy: []fsm.NodeID{localID},
	}
	locallyAssignedHallOrders[floor] = [] requestConsensus.Req {inactiveReq, inactiveReq}
}


 

func OrderConsensus(localID fsm.NodeID,
	numFloors int, 
	NewOrderChan <-chan elevio.ButtonEvent,
	CompletedOrderChan <-chan int, 
	PeersListUpdateChan <-chan [] fsm.NodeID,
	RemoteHallOrdersChan <-chan [][] requestConsensus.Req,
	ConfirmedHallOrdersToIOChan chan<- [][] bool,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	LocalHallOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var locallyAssignedHallOrders = make([][] requestConsensus.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)
	peersList := [] fsm.NodeID{}

// Initialize all to unknown
	for floor := range locallyAssignedHallOrders {
		for orderReq := range locallyAssignedHallOrders[floor] {

			locallyAssignedHallOrders[floor][orderReq] = requestConsensus.Req {
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
			fmt.Println("Setting order in local order matrix!")
			locallyAssignedHallOrders[a.Floor][a.Button] = requestConsensus.Req {
				state: PendingAck,
				ackBy: []fsm.NodeID{localID},
			}
			//Update network	

		case a := <- CompletedOrderChan:
			inactiveReq := requestConsensus.Req {
				state: Inactive, 
				ackBy: []fsm.NodeID{localID},
			}
			locallyAssignedHallOrders[a] = [] requestConsensus.Req {inactiveReq, inactiveReq}
			//Update optimal assigner
			//Update IO
			//Update network
		
		case a := <- PeersListUpdateChan:
			peersList = requestConsensus.UniqueIDSlice(a)

			if len(peersList) <= 1 {
				for floor := range locallyAssignedHallOrders {
					for orderReq := range locallyAssignedHallOrders[floor] {

						if locallyAssignedHallOrders[floor][orderReq].state == Inactive{
							locallyAssignedHallOrders[floor][orderReq].state = Unknown
						}
					}
				}
						
			}



		case a := <- RemoteHallOrdersChan:
			remoteHallOrders := a

			for floor := range locallyAssignedHallOrders {
				for orderReq := range locallyAssignedHallOrders[floor]{

					pLocal := &locallyAssignedHallOrders[floor][orderReq]
					pRemote := &remoteHallOrders[floor][orderReq]

					requestConsensus.merge(p_local, p_remote, localID, peersList)

	}

}