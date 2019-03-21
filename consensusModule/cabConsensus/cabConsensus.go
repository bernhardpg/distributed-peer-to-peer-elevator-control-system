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
	PeersListUpdateCabChan <-chan [] fsm.NodeID,
	LostNodeChan <-chan fsm.NodeID,
	RemoteCabOrdersChan <-chan map[fsm.NodeID] [] requestConsensus.Req,
	TurnOffCabLightChan chan<- int,
	TurnOnCabLightChan chan<- int,
	LocalCabOrdersToIOChan chan<- [] bool,
	ConfirmedCabOrdersToAssignerChan chan<- map[fsm.NodeID] [] bool,
	CabOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var localCabOrders = make(map[fsm.NodeID] [] requestConsensus.Req)
	var confirmedCabOrders = make(map[fsm.NodeID] [] bool)


	
	localCabOrders[localID] = make([] generalConsensusModule.Req, numFloors)
	confirmedCabOrders[localID] = make([] bool, numFloors)

	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = generalConsensusModule.Req {
				state: Inactive,
				ackBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}

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

		case a := <- CompletedCabOrderChan:
			
			localCabOrders[localID][a] = generalConsensusModule.Req {
					state: Inactive,
					ackBy: []fsm.NodeID{localID},
			}

			updateConfirmedCabOrders(localCabOrders, &confirmedCabOrders, TurnOffCabLightChan, TurnOnCabLightChan)
			ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			CabOrdersToNewtorkChan <- localCabOrders
			//HVA MED HALL ORDERS HER?

		case a := PeersListUpdateCabChan:
			peersList = generalConsensusModule.UniqueIDSlice(a)

		case a := LostNodeChan:
			
			//Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for orderReq := range reqArr{
						//If previous state was Inactive, change to Unknown
					if reqArr[orderReq].state == Inactive{
						localCabOrders[a][orderReq].state = Unknown
					}
				}
			}

		case a := RemoteCabOrdersChan:
			remoteCabOrders := a

			for remoteID, _ := range remoteCabOrders{
				reqArr, ok := localCabOrders[remoteID]

				//Add Node in local map if doesn't exist
				if !ok{

					localCabOrders[remoteID] = make([] generalConsensusModule.Req, numFloors)
					confirmedCabOrders[remoteID] = make([] bool, numFloors)

					for floor := range localCabOrders[remoteID] {
						localCabOrders[remoteID][floor] = generalConsensusModule.Req {
							state: Unknown,
							ackBy: nil,
						}
						confirmedCabOrders[remoteID][floor] = false

					}
				}

				newConfirmedOrInactiveFlag := false

				for orderReq := range reqArr {
					pLocal := &localCabOrders[remoteID][orderReq]
					remote := remoteCabOrders[remoteID][orderReq]

					newConfirmedOrInactiveFlag = generalConsensusModule.merge(pLocal, remote, localID, peersList)
				}

				    			
    		}
    		if newConfirmedOrInactiveFlag{
					updateConfirmedCabOrders(localCabOrders, &confirmedCabOrders, TurnOffCabLightsChan, TurnOnCabLightChan)
					ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
				}

			CabOrdersToNewtorkChan <- localCabOrders
}

}

}