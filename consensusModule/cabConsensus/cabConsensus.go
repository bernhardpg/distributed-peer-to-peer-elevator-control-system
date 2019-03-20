package cabConsensus

import(
	"fmt"
	"../../fsm"
	"../../elevio"
	"../requestConsensus"
)

func HallOrderConsensus(localID fsm.NodeID,
	numFloors int, 
	NewCabOrderChan <-chan elevio.ButtonEvent,
	CompletedCabOrderChan <-chan int,
	LostNodeChan <-chan fsm.NodeID,
	PeersListUpdateCabChan <-chan [] fsm.NodeID,
	RemoteCabOrdersChan <-chan [][] requestConsensus.Req,
	LocalCabOrdersToIOChan chan<- [] bool,
	ConfirmedCabOrdersToAssignerChan chan<- [][] bool,
	CabOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var locallyAssignedCabOrders = make([][] requestConsensus.Req, numFloors)
	var confirmedLocalCabOrders = make([] bool, numFloors)

	peersList := [] fsm.NodeID{}

	//Make map?


	for floor := range currHallOrdersChan {
		for orderType := range currHallOrdersChan[floor] {
			currHallOrdersChan[floor][orderType] = false
		}
		currCabOrdersChan[floor] = false
	}


	}
