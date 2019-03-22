package cabConsensus

import(
	"fmt"
	"../../datatypes"
	"../../elevio"
	"../generalConsensusModule"
)

type Channels struct {
	NewOrderChan chan int
	CompletedOrderChan chan int
}

func updateConfirmedCabOrders(
	localCabOrders map[datatypes.NodeID] [] datatypes.Req, 
	confirmedCabOrders map[datatypes.NodeID][] bool,
	localID datatypes.NodeID, 
	TurnOffCabLightChan chan<- elevio.ButtonEvent, 
	TurnOnCabLightChan chan<- elevio.ButtonEvent){

// TODO update new nodes in confirmedCabOrders 
 
	for cabID, _ := range localCabOrders{
		for floor := range localCabOrders[cabID]{

			if localCabOrders[cabID][floor].State == datatypes.Confirmed {

				//Set light if this node and not already set
				if (cabID == localID) && !confirmedCabOrders[cabID][floor] {
					setCabLight(floor, TurnOnCabLightChan)}	

				confirmedCabOrders[cabID][floor] = true

			} else {
				//Clear lights if not already cleared
				if (localCabOrders[cabID][floor].State == datatypes.Inactive) && (confirmedCabOrders[cabID][floor] == true){
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
	localID datatypes.NodeID,
	numFloors int, 
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeersListUpdateCabChan <-chan [] datatypes.NodeID,
	LostNodeChan <-chan datatypes.NodeID,
	RemoteCabOrdersChan <-chan map[datatypes.NodeID] [] datatypes.Req,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	ConfirmedCabOrdersToAssignerChan chan<- map[datatypes.NodeID] [] bool,
	CabOrdersToNetworkChan chan<- map[datatypes.NodeID] [] datatypes.Req) {

	var localCabOrders = make(map[datatypes.NodeID] [] datatypes.Req)
	var confirmedCabOrders = make(map[datatypes.NodeID] [] bool)

	peersList := [] datatypes.NodeID{}

	
	localCabOrders[localID] = make([] datatypes.Req, numFloors)
	confirmedCabOrders[localID] = make([] bool, numFloors)

	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req {
				State: datatypes.Unknown,
				AckBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}


	fmt.Println("\n cabConsensusModule initialized")

	for{
		select{

		case a := <- NewCabOrderChan:

			localCabOrders[localID][a] = datatypes.Req {
					State: datatypes.PendingAck,
					AckBy: []datatypes.NodeID{localID},
			}


			CabOrdersToNetworkChan <- localCabOrders

		case a := <- CompletedCabOrderChan:
			
			localCabOrders[localID][a] = datatypes.Req {
					State: datatypes.Inactive,
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
					if reqArr[floor].State == datatypes.Inactive{
						localCabOrders[a][floor].State = datatypes.Unknown
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