package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../generalConsensusModule"
	"../../network"
	)

type Channels struct {
	NewOrderChan chan elevio.ButtonEvent
	CompletedOrderChan chan int
}

func updateConfirmedHallOrders(
	localHallOrders [][] generalConsensusModule.Req, 
	confirmedHallOrders *[][] bool, 
	TurnOffHallLightChan chan<- elevio.ButtonEvent, 
	TurnOnHallLightChan chan<- elevio.ButtonEvent) {


	for floor := range localHallOrders {
		for orderType := range localHallOrders[floor] {

			if localHallOrders[floor][orderType].State == generalConsensusModule.Confirmed {
				//Set light if not already set
				if !(*confirmedHallOrders)[floor][orderType] {
					setHallLight(floor, orderType, TurnOnHallLightChan)}	

				(*confirmedHallOrders)[floor][orderType] = true
			} else {
				//Clear lights if not already cleared
				if (localHallOrders[floor][orderType].State == generalConsensusModule.Inactive) && ((*confirmedHallOrders)[floor][orderType] == true) {
					clearHallLights(floor, TurnOffHallLightChan) 
				}
				(*confirmedHallOrders)[floor][orderType] = false
			}			
		}		
	}
}

func setHallLight(currFloor int, orderType int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {
	buttonDir := (elevio.ButtonType)(orderType)

	buttonToIlluminate := elevio.ButtonEvent {
		Floor: currFloor,
		Button: buttonDir,
	}
	TurnOnHallLightChan <- buttonToIlluminate
}

func clearHallLights(currFloor int, TurnOffHallLightChan chan<- elevio.ButtonEvent) {


	callUpAtFloor := elevio.ButtonEvent {
		Floor: currFloor,
		Button: elevio.BT_HallUp,
	}

	callDownAtFloor := elevio.ButtonEvent {
		Floor: currFloor,
		Button: elevio.BT_HallDown,
	}

	TurnOffHallLightChan <- callDownAtFloor
	TurnOffHallLightChan <- callUpAtFloor	
}


func HallOrderConsensus(
	localID network.NodeID,
	numFloors int, 
	NewHallOrderChan <-chan elevio.ButtonEvent,
	CompletedHallOrderChan <-chan int, 
	PeersListUpdateHallChan <-chan [] network.NodeID,
	RemoteHallOrdersChan <-chan [][] generalConsensusModule.Req,
	TurnOffHallLightChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	ConfirmedHallOrdersToAssignerChan chan<- [][] bool,
	HallOrdersToNetworkChan chan<- [][] generalConsensusModule.Req) {

	var localHallOrders = make([][] generalConsensusModule.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)

	peersList := [] network.NodeID{}


	// Initialize all localHallOrders to unknown
	// (To allow overrides from remote data from other nodes on network)
	for floor := range localHallOrders {
		localHallOrders[floor] = make([] generalConsensusModule.Req, 2)

		for orderReq := range localHallOrders[floor] {	
			localHallOrders[floor][orderReq] = generalConsensusModule.Req {
				State: generalConsensusModule.Unknown,
				AckBy: nil,
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
			if len(peersList) <= 1 {
				break
			}

			// Make sure to never access elements outside of array
			if (a.Button == elevio.BT_HallUp || a.Button == elevio.BT_HallDown) {

				localHallOrders[a.Floor][a.Button] = generalConsensusModule.Req {
					State: generalConsensusModule.PendingAck,
					AckBy: [] network.NodeID { localID },
				}

				// Send updates to network module
				HallOrdersToNetworkChan <- localHallOrders
			}


		// Mark completed orders as inactive and update network module and optimalAssigner
		case a := <- CompletedHallOrderChan:

			inactiveReq := generalConsensusModule.Req {
				State: generalConsensusModule.Inactive, 
				// Delete ackBy list when transitioning to inactive
				AckBy: nil,
			}

			localHallOrders[a] = [] generalConsensusModule.Req { inactiveReq, inactiveReq }

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, TurnOffHallLightChan, TurnOnHallLightChan)
			
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

						if localHallOrders[floor][orderReq].State == generalConsensusModule.Inactive {
							localHallOrders[floor][orderReq].State = generalConsensusModule.Unknown
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

					newConfirmedOrInactiveFlag = generalConsensusModule.Merge(pLocal, remote, localID, peersList)
				}
			}

			// Only update confirmedHallOrders when orders are changed to inactive or confirmed
			if newConfirmedOrInactiveFlag {
				updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders, TurnOffHallLightChan, TurnOnHallLightChan)
				ConfirmedHallOrdersToAssignerChan <- confirmedHallOrders
			}

			// Update network module with new data
			HallOrdersToNetworkChan <- localHallOrders
		}
	}
}

