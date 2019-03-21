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
	RemoteCabOrdersChan <-chan map[fsm.NodeID] [] requestConsensus.Req,
	LocalCabOrdersToIOChan chan<- [] bool,
	ConfirmedCabOrdersToAssignerChan chan<- map[fsm.NodeID] [] bool,
	CabOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var locallyAssignedCabOrders = make(map[fsm.NodeID] [] requestConsensus.Req)
	var confirmedCabOrders = make(map[fsm.NodeID] [] bool)

	peersList := [] fsm.NodeID{}

	fmt.Println("\n cabConsensusModule initialized")


	for{
		select{

		case a := <- 
		}
	}


	}
