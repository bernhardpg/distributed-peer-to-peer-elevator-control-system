package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../../stateHandler"
	"../../fsm"
	"../generalConsensusModule"
	)

/*
func setLocalHallOrder(localID fsm.NodeID, buttonPress elevio.ButtonEvent, localHallOrders [][]requestConsensus.Req) {
	fmt.Println("Setting order in local order matrix!")

	localHallOrders[buttonPress.Floor][buttonPress.Button] = requestConsensus.Req {
		state: PendingAck,
		ackBy: []fsm.NodeID{localID},
	}
}
	// TODO set lights
	// TODO send orders to fsm!


func clearOrdersAtFloor(localID fsm.NodeID, floor int, localHallOrders [][] requestConsensus.Req, confirmedHallOrders [][] bool) {
	inactiveReq := requestConsensus.Req {
		state: Inactive, 
		ackBy: []fsm.NodeID{localID},
	}
	localHallOrders[floor] = [] requestConsensus.Req {inactiveReq, inactiveReq}
}

*/
			
func updateConfirmedHallOrders(localHallOrders [][] requestConsensus.Req, confirmedHallOrders *[][] bool){
	for floor := range localHallOrders {
		for orderReq := range localHallOrders[floor] {
			if localHallOrders[floor][orderReq].state == Confirmed {
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
	ConfirmedHallOrders chan<- [][] bool,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	HallOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var localHallOrders = make([][] requestConsensus.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)
	peersList := [] fsm.NodeID{}

// Initialize all to unknown

	for floor := range localHallOrders {
		localHallOrders[floor] = make([] requestConsensus.Req, 2)

		for orderReq := range localHallOrders[floor] {
			
			localHallOrders[floor][orderReq] = requestConsensus.Req {
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
			localHallOrders[a.Floor][a.Button] = requestConsensus.Req {
				state: PendingAck,
				ackBy: []fsm.NodeID{localID},
			}

			HallOrdersToNewtorkChan <- localHallOrders

			//Update network	

		case a := <- CompletedHallOrderChan:
			inactiveReq := requestConsensus.Req {
				state: Inactive, 
				ackBy: []fsm.NodeID{localID},
			}

			localHallOrders[a] = [] requestConsensus.Req {inactiveReq, inactiveReq}

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)
			ConfirmedHallOrdersToIOChan <- confirmedHallOrders
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			HallOrdersToNewtorkChan <- localHallOrders
			//Update IO
			//Update optimal assigner
			//Update network
		
		case a := <- PeersListUpdateHallChan:
			peersList = requestConsensus.UniqueIDSlice(a)

			if len(peersList) <= 1 {
				for floor := range localHallOrders {
					for orderReq := range localHallOrders[floor] {

						if localHallOrders[floor][orderReq].state == Inactive{
							localHallOrders[floor][orderReq].state = Unknown
						}
					}
				}
						
			}
			HallOrdersToNewtorkChan <- localHallOrders


		case a := <- RemoteHallOrdersChan:
			remoteHallOrders := a

			for floor := range localHallOrders {
				for orderReq := range localHallOrders[floor]{

					pLocal := &localHallOrders[floor][orderReq]
					remote := remoteHallOrders[floor][orderReq]

					requestConsensus.merge(pLocal, remote, localID, peersList)
				}
			}

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)
			ConfirmedHallOrdersToIOChan <- confirmedHallOrders
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			HallOrdersToNewtorkChan <- localHallOrders
		}
	}
}

