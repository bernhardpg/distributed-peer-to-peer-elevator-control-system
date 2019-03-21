package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../../stateHandler"
	"../../fsm"
	"../generalConsensusModule"
	)

			
func updateConfirmedHallOrders(localHallOrders [][] generalConsensusModule.Req, 
	confirmedHallOrders *[][] bool, 
	TurnOffHallLightsChan chan<- elevio.ButtonEvent, 
	TurnOnHallLightChan chan<- elevio.ButtonEvent){

	for floor := range localHallOrders {
		for dir := range localHallOrders[floor] {

			if localHallOrders[floor][dir].state == Confirmed {
				//Set light if not already set
				if !(*confirmedHallOrders)[floor][dir]{
					setHallLight(floor, dir, TurnOnLightChan)}	

				(*confirmedHallOrders)[floor][dir] = true
			}else{
				//Clear lights if not already cleared
				if (localHallOrders[floor][dir].state == Inactive) && ((*confirmedHallOrders)[floor][dir] == true){
					clearHallLights(floor, TurnOffLightsChan chan<- elevio.ButtonEvent) 
				}
				(*confirmedHallOrders)[floor][dir] = false
			}			
		}		
	}
}

func setHallLight(nFloor int, dir int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor: nFloor,
		Button: dir,
	}
	TurnOnLightsChan <- buttonToIlluminate
}

func clearHallLights(nFloor int, TurnOffLightsChan chan<- elevio.ButtonEvent){

	callUpAtFloor := elevio.ButtonEvent{
		Floor: nFloor,
		Button: BT_HallUp,
	}

	callDownAtFloor := elevio.ButtonEvent{
		Floor: nFloor,
		Button: BT_HallDown,
	}

	TurnOffLightsChan <- callDownAtFloor
	TurnOffLightsChan <- callUpAtFloor	
}


func HallOrderConsensus(localID fsm.NodeID,
	numFloors int, 
	NewHallOrderChan <-chan elevio.ButtonEvent,
	CompletedHallOrderChan <-chan int, 
	PeersListUpdateHallChan <-chan [] fsm.NodeID,
	RemoteHallOrdersChan <-chan [][] generalConsensusModule.Req,
	TurnOffHallLightsChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	HallOrdersToNewtorkChan chan<- [][] generalConsensusModule.Req) {

	var localHallOrders = make([][] generalConsensusModule.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)
	peersList := [] fsm.NodeID{}

// Initialize all to unknown

	for floor := range localHallOrders {
		localHallOrders[floor] = make([] generalConsensusModule.Req, 2)

		for orderReq := range localHallOrders[floor] {	
			localHallOrders[floor][orderReq] = generalConsensusModule.Req {
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
			//guard
			if (a.Button == BT_HallUp || a.Button == BT_HallDown){

				localHallOrders[a.Floor][a.Button] = generalConsensusModule.Req {
					state: PendingAck,
					ackBy: []fsm.NodeID{localID},
				}

				HallOrdersToNewtorkChan <- localHallOrders
			}

			//Update network	

		case a := <- CompletedHallOrderChan:
			inactiveReq := generalConsensusModule.Req {
				state: Inactive, 
				ackBy: []fsm.NodeID{localID},
			}

			localHallOrders[a] = [] generalConsensusModule.Req {inactiveReq, inactiveReq}

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, TurnOffHallLightsChan, TurnOnHallLightChan)
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			HallOrdersToNewtorkChan <- localHallOrders
			//Update IO
			//Update optimal assigner
			//Update network
		
		case a := <- PeersListUpdateHallChan:
			peersList = generalConsensusModule.UniqueIDSlice(a)

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
			newConfirmedOrInactiveFlag := false

			for floor := range localHallOrders {
				for orderReq := range localHallOrders[floor]{

					pLocal := &localHallOrders[floor][orderReq]
					remote := remoteHallOrders[floor][orderReq]

					newConfirmedOrInactiveFlag = generalConsensusModule.merge(pLocal, remote, localID, peersList)
				}
			}
			if newConfirmedOrInactiveFlag{
				updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, TurnOffHallLightsChan, TurnOnHallLightChan)
				ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			}

			HallOrdersToNewtorkChan <- localHallOrders
		}
	}
}

