package cabConsensus

import(
	"fmt"
	"../../fsm"
	"../../elevio"
	"../generalConsensusModule"
)


func updateConfirmedCabOrders(
	localCabOrders map[fsm.NodeID] [] requestConsensus.Req, 
	confirmedCabOrders map[fsm.NodeID] *[] bool,
	localID fsm.NodeID, 
	TurnOffCabLightChan chan<- elevio.ButtonEvent, 
	TurnOnCabLightChan chan<- elevio.ButtonEvent){

// TODO update new nodes in confirmedCabOrders 
 
	for cabID, _ := range localCabOrders{
		for floor := range localCabOrders[cabID]

			if localCabOrders[cabID][floor].state == Confirmed {

				//Set light if this node and not already set
				if (cabID == localID) && !(*confirmedCabOrders)[cabID][floor]{
					setCabLight(floor, TurnOnCabLightChan)}	

				(*confirmedCabOrders)[cabID][floor] = true

			}else{
				//Clear lights if not already cleared
				if (localCabOrders[cabID][floor].state == Inactive) && ((*confirmedCabOrders)[cabID][floor] == true){
					if {cabID == localID
						clearCabLight(floor, TurnOffCabLightChan) 
					}
				}
				(*confirmedHallOrders)[cabID][floor] = false
			}			
		}		
	}

func setCabLight(currFloor int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor: currFloor,
		Button: BT_Cab,
	}

	TurnOnCabLightChan <- buttonToIlluminate
}

func clearCabLights(currFloor int, TurnOffLightsChan chan<- elevio.ButtonEvent){

	callUpAtFloor := elevio.ButtonEvent{
		Floor: currFloor,
		Button: BT_Cab,
	}

	TurnOffCabLightChan <- callDownAtFloor
}



func CabOrderConsensus(
	localID fsm.NodeID,
	numFloors int, 
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeersListUpdateCabChan <-chan [] fsm.NodeID,
	LostNodeChan <-chan fsm.NodeID,
	RemoteCabOrdersChan <-chan map[fsm.NodeID] [] requestConsensus.Req,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	LocalCabOrdersToIOChan chan<- [] bool,
	ConfirmedCabOrdersToAssignerChan chan<- map[fsm.NodeID] [] bool,
	CabOrdersToNewtorkChan chan<- [][] requestConsensus.Req) {

	var localCabOrders = make(map[fsm.NodeID] [] requestConsensus.Req)
	var confirmedCabOrders = make(map[fsm.NodeID] [] bool)

	peersList := [] fsm.NodeID{}

	
	localCabOrders[localID] = make([] generalConsensusModule.Req, numFloors)
	confirmedCabOrders[localID] = make([] bool, numFloors)

	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = generalConsensusModule.Req {
				state: Unknown,
				ackBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}


	fmt.Println("\n cabConsensusModule initialized")

	for{
		select{

		case a := <- NewCabOrderChan:
			localCabOrders[localID][a] = generalConsensusModule.Req {
					state: PendingAck,
					ackBy: []fsm.NodeID{localID},
			}


			CabOrdersToNetworkChan <- localCabOrders

		case a := <- CompletedCabOrderChan:
			
			localCabOrders[localID][a] = generalConsensusModule.Req {
					state: Inactive,
					ackBy: nil,
			}

			// TODO clear lights here
 
			updateConfirmedCabOrders(localCabOrders, &confirmedCabOrders, TurnOffCabLightChan, TurnOnCabLightChan)
			ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			CabOrdersToNewtorkChan <- localCabOrders
			

		case a := PeersListUpdateCabChan:
			peersList = generalConsensusModule.UniqueIDSlice(a)

		case a := LostNodeChan:
			
			//Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr{
						//If previous state was Inactive, change to Unknown
					if reqArr[floor].state == Inactive{
						localCabOrders[a][floor].state = Unknown
					}
				}
			}

		case a := RemoteCabOrdersChan:
			remoteCabOrders := a

			for remoteID, _ := range remoteCabOrders{
				_, ok := localCabOrders[remoteID]

				//Add Node in local map if doesn't exist
				if !ok {

					localCabOrders[remoteID] = remoteCabOrders[remoteID]
					continue
					
				}

				reqArr := localCabOrders[remoteID]

				newConfirmedOrInactiveFlag := false

				for floor := range reqArr {
					pLocal := &localCabOrders[remoteID][floor]
					remote := remoteCabOrders[remoteID][floor]

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