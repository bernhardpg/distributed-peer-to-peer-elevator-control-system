package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../../stateHandler"
	"../../fsm"
	"../requestConsensus"
	)

/*
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

*/
			
func updateConfirmedHallOrders(locallyAssignedHallOrders [][] requestConsensus.Req, confirmedHallOrders *[][] bool){
	for floor := range locallyAssignedHallOrders {
		for orderReq := range locallyAssignedHallOrders[floor] {
			if locallyAssignedHallOrders[floor][orderReq].state == Confirmed {
				(*confirmedHallOrders)[floor][orderReq] = true
			}else{
				(*confirmedHallOrders)[floor][orderReq] = false
			}
			
		}
	}
}

func HallOrderConsensus(localID fsm.NodeID,
	numFloors int, 
	NewHallOrderChan <-chan elevio.ButtonEvent,
	CompletedHallOrderChan <-chan int, 
	PeersListUpdateHallChan <-chan [] fsm.NodeID,
	RemoteHallOrdersChan <-chan [][] requestConsensus.Req,
	ConfirmedHallOrdersToIOChan chan<- [][] bool,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	HallOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var locallyAssignedHallOrders = make([][] requestConsensus.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)
	peersList := [] fsm.NodeID{}

// Initialize all to unknown

	for floor := range locallyAssignedHallOrders {
		locallyAssignedHallOrders[floor] = make([] requestConsensus.Req, 2)

		for orderReq := range locallyAssignedHallOrders[floor] {
			
			locallyAssignedHallOrders[floor][orderReq] = requestConsensus.Req {
				state: Unknown,
				ackBy: nil,
			}

			confirmedHallOrders[floor] = [] bool{false, false}
		}
	}

	fmt.Println("\n hallConsensusModule initialized")

	for {

		select{

		case a := <- NewHallOrderChan:
			locallyAssignedHallOrders[a.Floor][a.Button] = requestConsensus.Req {
				state: PendingAck,
				ackBy: []fsm.NodeID{localID},
			}

			HallOrdersToNewtorkChan <- locallyAssignedHallOrders

			//Update network	

		case a := <- CompletedHallOrderChan:
			inactiveReq := requestConsensus.Req {
				state: Inactive, 
				ackBy: []fsm.NodeID{localID},
			}

			locallyAssignedHallOrders[a] = [] requestConsensus.Req {inactiveReq, inactiveReq}

			updateConfirmedHallOrders(locallyAssignedHallOrders, &confirmedHallOrders)
			ConfirmedHallOrdersToIOChan <- confirmedHallOrders
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			HallOrdersToNewtorkChan <- locallyAssignedHallOrders
			//Update IO
			//Update optimal assigner
			//Update network
		
		case a := <- PeersListUpdateHallChan:
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
			HallOrdersToNewtorkChan <- locallyAssignedHallOrders


		case a := <- RemoteHallOrdersChan:
			remoteHallOrders := a

			for floor := range locallyAssignedHallOrders {
				for orderReq := range locallyAssignedHallOrders[floor]{

					pLocal := &locallyAssignedHallOrders[floor][orderReq]
					pRemote := &remoteHallOrders[floor][orderReq]

					requestConsensus.merge(pLocal, pRemote, localID, peersList)
				}
			}

			updateConfirmedHallOrders(locallyAssignedHallOrders, &confirmedHallOrders)
			ConfirmedHallOrdersToIOChan <- confirmedHallOrders
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			HallOrdersToNewtorkChan <- locallyAssignedHallOrders
		}
	}
}

