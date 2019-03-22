package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../../stateHandler"
	"../../fsm"
	"../generalConsensusModule"
	)

			
func updateConfirmedHallOrders(
	localHallOrders [][] generalConsensusModule.Req, 
	confirmedHallOrders *[][] bool, 
	TurnOffHallLightChan chan<- elevio.ButtonEvent, 
	TurnOnHallLightChan chan<- elevio.ButtonEvent) {


	for floor := range localHallOrders {
		for dir := range localHallOrders[floor] {

			if localHallOrders[floor][dir].state == Confirmed {
				//Set light if not already set
				if !(*confirmedHallOrders)[floor][dir]{
					setHallLight(floor, dir, TurnOnHallLightChan)}	

				(*confirmedHallOrders)[floor][dir] = true
			}else{
				//Clear lights if not already cleared
				if (localHallOrders[floor][dir].state == Inactive) && ((*confirmedHallOrders)[floor][dir] == true){
					clearHallLights(floor, TurnOffHallLightChan chan<- elevio.ButtonEvent) 
				}
				(*confirmedHallOrders)[floor][dir] = false
			}			
		}		
	}
}

func setHallLight(currFloor int, dir int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor: currFloor,
		Button: dir,
	}
	TurnOnHallLightChan <- buttonToIlluminate
}

func clearHallLights(currFloor int, TurnOffHallLightChan chan<- elevio.ButtonEvent){

	callUpAtFloor := elevio.ButtonEvent{
		Floor: currFloor,
		Button: BT_HallUp,
	}

	callDownAtFloor := elevio.ButtonEvent{
		Floor: currFloor,
		Button: BT_HallDown,
	}

	TurnOffHallLightChan <- callDownAtFloor
	TurnOffHallLightChan <- callUpAtFloor	
}


func HallOrderConsensus(
	localID fsm.NodeID,
	numFloors int, 
	NewHallOrderChan <-chan elevio.ButtonEvent,
	CompletedHallOrderChan <-chan int, 
	PeersListUpdateHallChan <-chan [] fsm.NodeID,
	RemoteHallOrdersChan <-chan [][] generalConsensusModule.Req,
	TurnOffHallLightChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	HallOrdersToNetworkChan chan<- [][] generalConsensusModule.Req) {

	var localHallOrders = make([][] generalConsensusModule.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)

	peersList := [] fsm.NodeID{}


	// Initialize all localHallOrders to unknown
	// (To allow overrides from remote data from other nodes on network)
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

	fmt.Println("(hallConsensusModule) Initialized")

	for {

		select {

		// Store new local orders as pendingAck and update network module
		case a := <- NewHallOrderChan:

			// Don't accept new hall orders when alone on network
			// (Otherwise inactive orders might override confirmed orders when reconnecting to network)
			if len(peersList <= 1) {
				break
			}

			// Make sure to never access elements outside of array
			if (a.Button == BT_HallUp || a.Button == BT_HallDown) {

				localHallOrders[a.Floor][a.Button] = generalConsensusModule.Req {
					state: PendingAck,
					ackBy: [] fsm.NodeID { localID },
				}

				// Send updates to network module
				HallOrdersToNetworkChan <- localHallOrders
			}


		// Mark completed orders as inactive and update network module and optimalAssigner
		case a := <- CompletedHallOrderChan:

			inactiveReq := generalConsensusModule.Req {
				state: Inactive, 
				// Delete ackBy list when transitioning to inactive
				ackBy: nil,
			}

			localHallOrders[a] = [] generalConsensusModule.Req { inactiveReq, inactiveReq }

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, localID fsm.NodeID, TurnOffHallLightChan, TurnOnHallLightChan)
			
			// Send updates to optimalAssigner
			ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			
			// Send updates to network module
			HallOrdersToNetworkChan <- localHallOrders
			

		// Received changes in peerlist from network module
		case a := <- PeersListUpdateHallChan:

			peersList = generalConsensusModule.UniqueIDSlice(a)

			// Set all inactive hall orders to unknown if alone on network
			if len(peersList) <= 1 {
				for floor := range localHallOrders {
					for orderReq := range localHallOrders[floor] {

						if localHallOrders[floor][orderReq].state == Inactive {
							localHallOrders[floor][orderReq].state = Unknown
						}
					}
				}
				
				// Inform network module that changes have been made
				HallOrdersToNetworkChan <- localHallOrders		
			}

		// Merge received remoteHallOrders from network module with local data in localHallOrders 
		case a := <- RemoteHallOrdersChan:

			remoteHallOrders := a

			newConfirmedOrInactiveFlag := false

			for floor := range localHallOrders {
				for orderReq := range localHallOrders[floor] {

					pLocal := &localHallOrders[floor][orderReq]
					remote := remoteHallOrders[floor][orderReq]

					newConfirmedOrInactiveFlag = generalConsensusModule.merge(pLocal, remote, localID, peersList)
				}
			}

			// Only update confirmedHallOrders when orders are changed to inactive or confirmed
			if newConfirmedOrInactiveFlag {
				updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, localID fsm.NodeID, TurnOffHallLightChan, TurnOnHallLightChan)
				ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			}

			// Update network module with new data
			HallOrdersToNetworkChan <- localHallOrders
		}
	}
}

