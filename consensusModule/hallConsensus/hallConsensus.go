package hallConsensus

import(
	"fmt"
	"../../elevio"
	"../generalConsensusModule"
	"../../nodeStatesHandler"
	)

type Channels struct {
	NewOrderChan chan elevio.ButtonEvent
	CompletedOrderChan chan int
	ConfirmedOrdersChan chan [][] bool
	LocalOrdersChan chan [][] generalConsensusModule.Req
	RemoteOrdersChan chan [][] generalConsensusModule.Req
}

type LocalHallOrdersMsg struct {
	ID nodeStatesHandler.NodeID
	HallOrders [][] generalConsensusModule.Req
}

// Constructs a matrix with boolean values for the confirmed orders
func updateConfirmedHallOrders(
	localHallOrders [][] generalConsensusModule.Req, 
	confirmedHallOrders *[][] bool) {
	//TurnOffHallLightChan chan<- elevio.ButtonEvent, 
	//TurnOnHallLightChan chan<- elevio.ButtonEvent) {

	for floor := range localHallOrders {
		for orderType := range localHallOrders[floor] {

			if localHallOrders[floor][orderType].State == generalConsensusModule.Confirmed {
				/*//Set light if not already set
				if !(*confirmedHallOrders)[floor][orderType] {
					setHallLight(floor, orderType, TurnOnHallLightChan)
				}*/	

				(*confirmedHallOrders)[floor][orderType] = true
			} else {
				/*//Clear lights if not already cleared
				if (localHallOrders[floor][orderType].State == generalConsensusModule.Inactive) && ((*confirmedHallOrders)[floor][orderType] == true) {
					clearHallLights(floor, TurnOffHallLightChan) 
				}*/
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


func ConsensusModule(
	localID nodeStatesHandler.NodeID,
	numFloors int, 
	NewOrderChan <-chan elevio.ButtonEvent,
	ConfirmedOrdersChan chan<- [][] bool,
	CompletedOrderChan <-chan int, 
	//peerlistUpdateHallChan <-chan [] nodeStatesHandler.NodeID,
	TurnOffHallLightChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	LocalOrdersChan chan<- [][] generalConsensusModule.Req,
	RemoteOrdersChan <-chan [][] generalConsensusModule.Req) {

	var localHallOrders = make([][] generalConsensusModule.Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)

	// TODO remove localID
	peerlist := [] nodeStatesHandler.NodeID { localID }


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
	
	// Send initialized variables
	// ------

	// Send initialized matrix to network module
	LocalOrdersChan <- localHallOrders

	// Create initial confirmedHallOrder matrix
	updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)
	// Send initialized matrix to optimalAssigner
	ConfirmedOrdersChan <- confirmedHallOrders


	fmt.Println("(hallConsensusModule) Initialized")


	// Handle consensus when new data enters system
	// ------

	for {

		select {

		// Store new local orders as pendingAck and update network module
		case a := <- NewOrderChan:

			/*// Don't accept new hall orders when alone on network
			// (Otherwise inactive orders might override confirmed orders when reconnecting to network)
			if len(peerlist) <= 1 {
				break
			}*/

			// Set order to pendingAck
			// (Make sure to never access elements outside of array)
			if (a.Button == elevio.BT_HallUp || a.Button == elevio.BT_HallDown) {

				localHallOrders[a.Floor][a.Button] = generalConsensusModule.Req {
					State: generalConsensusModule.PendingAck,
					AckBy: [] nodeStatesHandler.NodeID { localID },
				}

				// Send updates to network module
				LocalOrdersChan <- localHallOrders
			}

		// Mark completed orders as inactive and update network module and optimalAssigner
		case a := <- CompletedOrderChan:

			// Set both dir Up and dir Down to inactive
			inactiveReq := generalConsensusModule.Req {
				State: generalConsensusModule.Inactive, 
				AckBy: nil, // Delete ackBy list when transitioning to inactive
			}
			localHallOrders[a] = [] generalConsensusModule.Req {
				inactiveReq,
				inactiveReq,
			}

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)

			// Send updates to optimalAssigner
			ConfirmedOrdersChan <- confirmedHallOrders
			
			// Send updates to network module
			LocalOrdersChan <- localHallOrders


		/*// Received changes in peerlist from network module
		case a := <- peerlistUpdateHallChan:

			peerlist = generalConsensusModule.UniqueIDSlice(a)

			// Set all inactive hall orders to unknown if alone on network
			if len(peerlist) <= 1 {
				for floor := range localHallOrders {
					for orderReq := range localHallOrders[floor] {

						if localHallOrders[floor][orderReq].State == generalConsensusModule.Inactive {
							localHallOrders[floor][orderReq].State = generalConsensusModule.Unknown
						}
					}
				}
				
				// Inform network module that changes have been made
				LocalOrdersChan <- localHallOrders		
			}*/

		// Merge received remoteHallOrders from network module with local data in localHallOrders 
		case a := <- RemoteOrdersChan:

			remoteHallOrders := a

			newConfirmedOrInactiveFlag := false

			// Merge world views for every order in HallOrder matrix
			for floor := range localHallOrders {
				for orderReq := range localHallOrders[floor] {

					pLocal := &localHallOrders[floor][orderReq]
					remote := remoteHallOrders[floor][orderReq]

					// Make flag stay true if set to true once
					newConfirmedOrInactiveFlag = newConfirmedOrInactiveFlag || generalConsensusModule.Merge(pLocal, remote, localID, peerlist)
				}
			}

			// Only update confirmedHallOrders when orders are changed to inactive or confirmed
			if newConfirmedOrInactiveFlag {
				updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)
				fmt.Printf("(hallConsensus) Address of confirmedHallOrders: %p\n", &confirmedHallOrders)

				ConfirmedOrdersChan <- confirmedHallOrders
			}

			// Update network module with new data
			LocalOrdersChan <- localHallOrders
		}
	}
}

