package hallConsensus

import(
	"fmt"
	"../../datatypes"
	"../../elevio"
	"../generalConsensusModule"
	)

type Channels struct {
	NewOrderChan chan elevio.ButtonEvent
	CompletedOrderChan chan int
	ConfirmedOrdersChan chan datatypes.ConfirmedHallOrdersMatrix
	LocalOrdersChan chan datatypes.HallOrdersMatrix
	RemoteOrdersChan chan datatypes.HallOrdersMatrix
}

type LocalHallOrdersMsg struct {
	ID datatypes.NodeID
	HallOrders datatypes.HallOrdersMatrix
}

// Constructs a matrix with boolean values for the confirmed orders
func updateConfirmedHallOrders(
	localHallOrders datatypes.HallOrdersMatrix, 
	confirmedHallOrders *datatypes.ConfirmedHallOrdersMatrix) {
	//TurnOffHallLightChan chan<- elevio.ButtonEvent, 
	//TurnOnHallLightChan chan<- elevio.ButtonEvent) {

	for floor := range localHallOrders {
		for orderType := range localHallOrders[floor] {

			if localHallOrders[floor][orderType].State == datatypes.Confirmed {
				/*//Set light if not already set
				if !(*confirmedHallOrders)[floor][orderType] {
					setHallLight(floor, orderType, TurnOnHallLightChan)
				}*/	

				(*confirmedHallOrders)[floor][orderType] = true
			} else {
				/*//Clear lights if not already cleared
				if (localHallOrders[floor][orderType].State == datatypes.Inactive) && ((*confirmedHallOrders)[floor][orderType] == true) {
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
	localID datatypes.NodeID,
	NewOrderChan <-chan elevio.ButtonEvent,
	ConfirmedOrdersChan chan<- datatypes.ConfirmedHallOrdersMatrix,
	CompletedOrderChan <-chan int, 
	//peerlistUpdateHallChan <-chan [] datatypes.NodeID,
	TurnOffHallLightChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	LocalOrdersChan chan<- datatypes.HallOrdersMatrix,
	RemoteOrdersChan <-chan datatypes.HallOrdersMatrix) {

	// All orders will be initialized to Unknown (zero-state)
	var localHallOrders datatypes.HallOrdersMatrix
	
	var confirmedHallOrders datatypes.ConfirmedHallOrdersMatrix

	// TODO remove localID
	peerlist := [] datatypes.NodeID { localID }
	
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

				localHallOrders[a.Floor][a.Button] = datatypes.Req {
					State: datatypes.PendingAck,
					AckBy: [] datatypes.NodeID { localID },
				}

				// Send updates to network module
				LocalOrdersChan <- localHallOrders
			}

		// Mark completed orders as inactive and update network module and optimalAssigner
		// with all confirmedHallOrders
		case a := <- CompletedOrderChan:

			// Set both dir Up and dir Down to inactive
			inactiveReq := datatypes.Req {
				State: datatypes.Inactive, 
				AckBy: nil, // Delete ackBy list when transitioning to inactive
			}
			localHallOrders[a] = [2] datatypes.Req {
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

						if localHallOrders[floor][orderReq].State == datatypes.Inactive {
							localHallOrders[floor][orderReq].State = datatypes.Unknown
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

				ConfirmedOrdersChan <- confirmedHallOrders
			}

			// Update network module with new data
			LocalOrdersChan <- localHallOrders
		}
	}
}

