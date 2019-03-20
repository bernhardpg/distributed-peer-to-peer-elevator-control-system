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


func OrderConsensus(localID fsm.NodeID,
	numFloors int, 
	CompletedOrderChan <-chan int, 
	NewOrderChan <-chan elevio.ButtonEvent,
	RemoteHallOrdersChan <-chan [][] Req,
	ConfirmedHallOrdersToIOChan chan<- [][] bool,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	LocalHallOrdersToNewtorkChan chan<- [][] Req) {

	var locallyAssignedHallOrders = make([][] Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)

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
		

		case a := <- RemoteHallOrdersChan:
			remoteHallOrders := a

			for floor := range locallyAssignedHallOrders {
				for orderReq := range locallyAssignedHallOrders[floor]{

					//Switching on local state
					switch locallyAssignedHallOrders[floor][orderReq].state {

					case Inactive:
						if (remoteHallOrders[floor][orderReq].state == PendingAck){
							locallyAssignedHallOrders[floor][orderReq] = Req {PendingAck, append(remoteHallOrders[floor][orderReq].ackBy, localID)}

						}


					case PendingAck:

					case Confirmed:

					case Unknown:

					}
				}

				
			}
		}

	}

}