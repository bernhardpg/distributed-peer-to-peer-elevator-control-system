package cabConsensus

import(
	"fmt"
	"../../fsm"
	"../../elevio"
	"../generalConsensusModule"
)

func CabOrderConsensus(localID fsm.NodeID,
	numFloors int, 
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	LostNodeChan <-chan fsm.NodeID,
	PeersListUpdateCabChan <-chan [] fsm.NodeID,
	RemoteCabOrdersChan <-chan map[fsm.NodeID] [] requestConsensus.Req,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	LocalCabOrdersToIOChan chan<- [] bool,
	ConfirmedCabOrdersToAssignerChan chan<- map[fsm.NodeID] [] bool,
	CabOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var localCabOrders = make(map[fsm.NodeID] [] requestConsensus.Req)
	var confirmedCabOrders = make(map[fsm.NodeID] [] bool)

	unknownReq := requestConsensus.Req {
		state: Unknown,
		ackBy: nil,
	}

	//Kan være at den bør init til unknown heller?
	localCabOrders[localID] = make([] generalConsensusModule.Req, numFloors)


	peersList := [] fsm.NodeID{}


	fmt.Println("\n cabConsensusModule initialized")


	for{
		select{

		case a := <- NewCabOrderChan:
			localCabOrders[localID][a] = generalConsensusModule.Req {
					state: PendingAck,
					ackBy: []fsm.NodeID{localID},
			}

			CabOrdersToNewtorkChan <- localCabOrders

			
}


		}
	}


	}
