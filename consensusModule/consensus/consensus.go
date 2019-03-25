package consensus

import (
	"../../datatypes"
	"../../elevio"
	"../consensusFns"
	"fmt"
)

// HallChannels ...
// Channels used for communication related to consensus of hall orders with other modules
type HallOrderChannels struct {
	NewOrderChan        chan elevio.ButtonEvent
	CompletedOrderChan  chan int
	ConfirmedOrdersChan chan datatypes.ConfirmedHallOrdersMatrix
	LocalOrdersChan     chan datatypes.HallOrdersMatrix
	RemoteOrdersChan    chan datatypes.HallOrdersMatrix
	PeerlistUpdateChan  chan []datatypes.NodeID
}

// LocalHallOrdersMsg ...
// Used for broadcasting localHallOrders to other nodes
type LocalHallOrdersMsg struct {
	ID         datatypes.NodeID
	HallOrders datatypes.HallOrdersMatrix
}

// updateConfirmedHallOrders ...
// Constructs a boolean matrix where only confirmed orders are set to true
func updateConfirmedHallOrders(
	localHallOrders datatypes.HallOrdersMatrix,
	confirmedHallOrders *datatypes.ConfirmedHallOrdersMatrix) {

	for floor := range localHallOrders {
		for orderType := range localHallOrders[floor] {
			if localHallOrders[floor][orderType].State == datatypes.Confirmed {
				(*confirmedHallOrders)[floor][orderType] = true
			} else {
				(*confirmedHallOrders)[floor][orderType] = false
			}
		}
	}
}

func setHallLight(currFloor int, orderType int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {
	buttonDir := (elevio.ButtonType)(orderType)

	buttonToIlluminate := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: buttonDir,
	}
	TurnOnHallLightChan <- buttonToIlluminate
}

func clearHallLights(currFloor int, TurnOffHallLightChan chan<- elevio.ButtonEvent) {

	callUpAtFloor := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_HallUp,
	}

	callDownAtFloor := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_HallDown,
	}

	TurnOffHallLightChan <- callDownAtFloor
	TurnOffHallLightChan <- callUpAtFloor
}

// HallConsensusModule ...
// Handles the information distribution for hall orders between nodes.
// Keeps track of which orders are currently confirmed by all nodes, which orders that are still pending acknowledgement,
// and which orders that are completed (inactive). Only confirmed orders are passed along to the optimal assigner, making
// sure that all nodes agree on the distribution of all of the orders at all times.
func HallConsensusModule(
	localID datatypes.NodeID,
	NewOrderChan <-chan elevio.ButtonEvent,
	ConfirmedOrdersChan chan<- datatypes.ConfirmedHallOrdersMatrix,
	CompletedOrderChan <-chan int,
	TurnOffHallLightChan chan<- elevio.ButtonEvent,
	TurnOnHallLightChan chan<- elevio.ButtonEvent,
	LocalOrdersChan chan<- datatypes.HallOrdersMatrix,
	RemoteOrdersChan <-chan datatypes.HallOrdersMatrix,
	PeerlistUpdateChan <-chan []datatypes.NodeID) {

	// Initialize variables
	// ----
	// All orders will be initialized to Unknown (zero-state)
	var localHallOrders datatypes.HallOrdersMatrix
	var confirmedHallOrders datatypes.ConfirmedHallOrdersMatrix
	peerlist := []datatypes.NodeID{}

	// Send initialized variables to other modules
	// ------
	// Send initialized matrix to network module
	LocalOrdersChan <- localHallOrders

	// Create initial confirmedHallOrder matrix
	updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)
	// Send initialized matrix to optimalAssigner
	ConfirmedOrdersChan <- confirmedHallOrders

	fmt.Println("(hallConsensusModule) Initialized")

	// Handle consensus logic when new data enters system
	// ------
	for {

		select {

		// Store new local orders as pendingAck and update network module
		case a := <-NewOrderChan:

			// Don't accept new hall orders when alone on network
			// (Otherwise inactive orders might override confirmed orders when reconnecting to network)
			/*if len(peerlist) <= 1 {
				break
			}*/

			// Set order to pendingAck
			// (Make sure to never access elements outside of array)
			if a.Button == elevio.BT_HallUp || a.Button == elevio.BT_HallDown {

				localHallOrders[a.Floor][a.Button] = datatypes.Req{
					State: datatypes.PendingAck,
					AckBy: []datatypes.NodeID{localID},
				}

				// Send updates to network module
				LocalOrdersChan <- localHallOrders

			}

		// Mark completed orders as inactive and update network module and optimalAssigner
		// with all confirmedHallOrders
		case a := <-CompletedOrderChan:

			clearHallLights(a, TurnOffHallLightChan)

			// Set both dir Up and dir Down to inactive
			inactiveReq := datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil, // Delete ackBy list when transitioning to inactive
			}
			localHallOrders[a] = [2]datatypes.Req{
				inactiveReq,
				inactiveReq,
			}

			updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)

			// Send updates to optimalAssigner
			ConfirmedOrdersChan <- confirmedHallOrders

			// Send updates to network module
			LocalOrdersChan <- localHallOrders

		// Received changes in peerlist from network module
		case a := <-PeerlistUpdateChan:

			peerlist = consensusFns.UniqueIDSlice(a)

			// Set all inactive hall orders to unknown if alone on network
			if len(peerlist) <= 1 {
				for floor := range localHallOrders {
					for orderType := range localHallOrders[floor] {

						if localHallOrders[floor][orderType].State == datatypes.Inactive {
							localHallOrders[floor][orderType].State = datatypes.Unknown
						}
					}
				}

				// Inform network module that changes have been made
				LocalOrdersChan <- localHallOrders
			}

		// Merge received remoteHallOrders from network module with local data in localHallOrders
		case a := <-RemoteOrdersChan:

			remoteHallOrders := a

			confirmedOrdersChangedFlag := false

			// Merge world views for every order in HallOrder matrix
			for floor := range localHallOrders {
				for orderType := range localHallOrders[floor] {

					pLocal := &localHallOrders[floor][orderType]
					remote := remoteHallOrders[floor][orderType]

					newInactiveFlag, newConfirmedFlag := consensusFns.Merge(pLocal, remote, localID, peerlist)

					// Make flag stay true if set to true once
					confirmedOrdersChangedFlag = confirmedOrdersChangedFlag || newInactiveFlag || newConfirmedFlag

					if newInactiveFlag {
						clearHallLights(floor, TurnOffHallLightChan)
					} else if newConfirmedFlag {
						setHallLight(floor, orderType, TurnOnHallLightChan)
					}
				}
			}

			// Only update confirmedHallOrders when orders are changed to inactive or confirmed
			if confirmedOrdersChangedFlag {
				updateConfirmedHallOrders(localHallOrders, &confirmedHallOrders)

				ConfirmedOrdersChan <- confirmedHallOrders
			}

			// Update network module with new data
			LocalOrdersChan <- localHallOrders
			fmt.Println(localHallOrders)
		}
	}
}
