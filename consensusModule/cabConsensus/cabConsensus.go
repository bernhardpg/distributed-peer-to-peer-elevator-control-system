package cabConsensus

import(
	"fmt"
	"../../elevio"
	"../../nodeStatesHandler"
	"../generalConsensusModule"

)

type Channels struct {
	NewOrderChan chan int
	CompletedOrderChan chan int
}

func updateConfirmedCabOrders(
	localCabOrders map[nodeStatesHandler.NodeID] [] generalConsensusModule.Req, 
	confirmedCabOrders map[nodeStatesHandler.NodeID][] bool,
	localID nodeStatesHandler.NodeID, 
	TurnOffCabLightChan chan<- elevio.ButtonEvent, 
	TurnOnCabLightChan chan<- elevio.ButtonEvent){

// TODO update new nodes in confirmedCabOrders 
 
	for cabID, _ := range localCabOrders{
		for floor := range localCabOrders[cabID]{

			if localCabOrders[cabID][floor].State == generalConsensusModule.Confirmed {

				//Set light if this node and not already set
				if (cabID == localID) && !confirmedCabOrders[cabID][floor] {
					setCabLight(floor, TurnOnCabLightChan)}	

				confirmedCabOrders[cabID][floor] = true

			} else {
				//Clear lights if not already cleared
				if (localCabOrders[cabID][floor].State == generalConsensusModule.Inactive) && (confirmedCabOrders[cabID][floor] == true){
					if cabID == localID {
						clearCabLight(floor, TurnOffCabLightChan) 
					}
				}
				confirmedCabOrders[cabID][floor] = false
			}			
		}		
	}
}

func setCabLight(currFloor int, TurnOnCabLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor: currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOnCabLightChan <- buttonToIlluminate
}

func clearCabLight(currFloor int, TurnOffCabLightChan chan<- elevio.ButtonEvent){

	buttonToClear := elevio.ButtonEvent{
		Floor: currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOffCabLightChan <- buttonToClear
}



func CabOrderConsensus(
	localID nodeStatesHandler.NodeID,
	numFloors int, 
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeersListUpdateCabChan <-chan [] nodeStatesHandler.NodeID,
	LostNodeChan <-chan nodeStatesHandler.NodeID,
	RemoteCabOrdersChan <-chan map[nodeStatesHandler.NodeID] [] generalConsensusModule.Req,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	ConfirmedCabOrdersToAssignerChan chan<- map[nodeStatesHandler.NodeID] [] bool,
	CabOrdersToNetworkChan chan<- map[nodeStatesHandler.NodeID] [] generalConsensusModule.Req) {

	var localCabOrders = make(map[nodeStatesHandler.NodeID] [] generalConsensusModule.Req)
	var confirmedCabOrders = make(map[nodeStatesHandler.NodeID] [] bool)

	peersList := [] nodeStatesHandler.NodeID{}

	
	localCabOrders[localID] = make([] generalConsensusModule.Req, numFloors)
	confirmedCabOrders[localID] = make([] bool, numFloors)

	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = generalConsensusModule.Req {
				State: generalConsensusModule.Unknown,
				AckBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}


	fmt.Println("\n cabConsensusModule initialized")

	for{
		select{

		case a := <- NewCabOrderChan:

			localCabOrders[localID][a] = generalConsensusModule.Req {
					State: generalConsensusModule.PendingAck,
					AckBy: []nodeStatesHandler.NodeID{localID},
			}


			CabOrdersToNetworkChan <- localCabOrders

		case a := <- CompletedCabOrderChan:
			
			localCabOrders[localID][a] = generalConsensusModule.Req {
					State: generalConsensusModule.Inactive,
					AckBy: nil,
			}

			// TODO clear lights here
 
			updateConfirmedCabOrders(localCabOrders, confirmedCabOrders, localID, TurnOffCabLightChan, TurnOnCabLightChan)
			ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			CabOrdersToNetworkChan <- localCabOrders
			

		case a := <- PeersListUpdateCabChan:
			peersList = generalConsensusModule.UniqueIDSlice(a)

		case a := <- LostNodeChan:
			
			//Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr{
						//If previous state was Inactive, change to Unknown
					if reqArr[floor].State == generalConsensusModule.Inactive{
						localCabOrders[a][floor].State = generalConsensusModule.Unknown
					}
				}
			CabOrdersToNetworkChan <- localCabOrders

			}

		case a := <- RemoteCabOrdersChan:
			remoteCabOrders := a

			newConfirmedOrInactiveFlag := false

			for remoteID, _ := range remoteCabOrders{
				_, ok := localCabOrders[remoteID]

				//Add Node in local map if doesn't exist
				if !ok {

					localCabOrders[remoteID] = remoteCabOrders[remoteID]
					continue
					
				}

				reqArr := localCabOrders[remoteID]


				for floor := range reqArr {
					pLocal := &localCabOrders[remoteID][floor]
					remote := remoteCabOrders[remoteID][floor]

					newConfirmedOrInactiveFlag = generalConsensusModule.Merge(pLocal, remote, localID, peersList)
				}

				    			
    		}

    		if newConfirmedOrInactiveFlag{
					updateConfirmedCabOrders(localCabOrders, confirmedCabOrders, localID, TurnOffCabLightChan, TurnOnCabLightChan)
					ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
				}

			CabOrdersToNetworkChan <- localCabOrders
		}

	}
}