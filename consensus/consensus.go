package consensus

import (
	"../datatypes"
	"../elevio"
	"fmt"
)

// merge ...
// Forms the basis for all the consensus logic.
// Merges the wordview of a single local order request with a single remote order request.
// Possible order states:
//		Unknown - Nothing can be said with certainty about the order, will get overriden by all other states
//		Inactive - The order is completed and hence to be regarded as inactive
//		PendingAck - The order is pending acknowledgement from the other nodes on the network before it can be handled by a node
//		Confirmed - The order is confirmed by all nodes on the network and is ready to be served by a node
// @return newConfirmedFlag: the order was set to Confirmed
// @return newInactiveFlag: the order was set to Inactive
func merge(
	pLocal *datatypes.Req,
	remote datatypes.Req,
	localID datatypes.NodeID,
	peersList []datatypes.NodeID) (bool, bool) {

	newConfirmedFlag := false
	newInactiveFlag := false

	switch (*pLocal).State {

	case datatypes.Inactive:
		if remote.State == datatypes.PendingAck {
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
			}
		}

	case datatypes.PendingAck:
		(*pLocal).AckBy = uniqueIDSlice(append(remote.AckBy, localID))

		if (remote.State == datatypes.Confirmed) || containsList((*pLocal).AckBy, peersList) {
			(*pLocal).State = datatypes.Confirmed
			newConfirmedFlag = true
		}

	case datatypes.Confirmed:
		(*pLocal).AckBy = uniqueIDSlice(append(remote.AckBy, localID))

		if remote.State == datatypes.Inactive {
			*pLocal = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}
			newInactiveFlag = true
		}

	case datatypes.Unknown:
		switch remote.State {

		case datatypes.Inactive:
			*pLocal = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}
			newInactiveFlag = true

		case datatypes.PendingAck:
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
			}

		case datatypes.Confirmed:
			*pLocal = datatypes.Req{
				State: datatypes.Confirmed,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
				//Signaliser datatypes.Confirmed
			}
			newConfirmedFlag = true

		}
	}

	return newInactiveFlag, newConfirmedFlag
}

// TODO what does this do??
func uniqueIDSlice(IDSlice []datatypes.NodeID) []datatypes.NodeID {

	keys := make(map[datatypes.NodeID]bool)
	list := []datatypes.NodeID{}

	for _, entry := range IDSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// containtsID ...
// Returns whether or not the NodeID list passed as the first argument contains the NodeID passed as the second param
func containsID(s []datatypes.NodeID, e datatypes.NodeID) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// containsList ...
//Returns true if primaryList contains listFraction
func containsList(primaryList []datatypes.NodeID, listFraction []datatypes.NodeID) bool {
	for _, a := range listFraction {
		if !containsID(primaryList, a) {
			return false
		}
	}
	return true
}

// ----------
// Setting the elevator lights
// ----------
// TODO generalize light functions

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

func setCabLight(currFloor int, TurnOnCabLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOnCabLightChan <- buttonToIlluminate
}

func clearCabLight(currFloor int, TurnOffCabLightChan chan<- elevio.ButtonEvent) {

	buttonToClear := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOffCabLightChan <- buttonToClear
}

// ----------
// Hall Order Consensus
// ----------

// HallOrderChannels ...
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

// HallOrdersModule ...
// Handles the information distribution for hall orders between nodes.
// Keeps track of which orders are currently confirmed by all nodes, which orders that are still pending acknowledgement,
// and which orders that are completed (Inactive). Only confirmed orders are passed along to the optimal assigner, making
// sure that all nodes agree on the distribution of all of the orders at all times.
func HallOrdersModule(
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

			peerlist = uniqueIDSlice(a)

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

					newInactiveFlag, newConfirmedFlag := merge(pLocal, remote, localID, peerlist)

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

// ----------
// Cab Order Consensus
// ----------

type CabOrderChannels struct {
	NewOrderChan       chan int
	CompletedOrderChan chan int
}

func updateConfirmedCabOrders(
	localCabOrders map[datatypes.NodeID][]datatypes.Req,
	confirmedCabOrders map[datatypes.NodeID][]bool,
	localID datatypes.NodeID,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent) {

	// TODO update new nodes in confirmedCabOrders

	for cabID, _ := range localCabOrders {
		for floor := range localCabOrders[cabID] {

			if localCabOrders[cabID][floor].State == datatypes.Confirmed {

				//Set light if this node and not already set
				if (cabID == localID) && !confirmedCabOrders[cabID][floor] {
					setCabLight(floor, TurnOnCabLightChan)
				}

				confirmedCabOrders[cabID][floor] = true

			} else {
				//Clear lights if not already cleared
				if (localCabOrders[cabID][floor].State == datatypes.Inactive) && (confirmedCabOrders[cabID][floor] == true) {
					if cabID == localID {
						clearCabLight(floor, TurnOffCabLightChan)
					}
				}
				confirmedCabOrders[cabID][floor] = false
			}
		}
	}
}

func CabOrderConsensus(
	localID datatypes.NodeID,
	numFloors int,
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeersListUpdateCabChan <-chan []datatypes.NodeID,
	LostNodeChan <-chan datatypes.NodeID,
	RemoteCabOrdersChan <-chan map[datatypes.NodeID][]datatypes.Req,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	ConfirmedCabOrdersToAssignerChan chan<- map[datatypes.NodeID][]bool,
	CabOrdersToNetworkChan chan<- map[datatypes.NodeID][]datatypes.Req) {

	var localCabOrders = make(map[datatypes.NodeID][]datatypes.Req)
	var confirmedCabOrders = make(map[datatypes.NodeID][]bool)

	peersList := []datatypes.NodeID{}

	localCabOrders[localID] = make([]datatypes.Req, numFloors)
	confirmedCabOrders[localID] = make([]bool, numFloors)

	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req{
			State: datatypes.Unknown,
			AckBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}

	fmt.Println("\n cabConsensusModule initialized")

	for {
		select {

		case a := <-NewCabOrderChan:

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: []datatypes.NodeID{localID},
			}

			CabOrdersToNetworkChan <- localCabOrders

		case a := <-CompletedCabOrderChan:

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}

			// TODO clear lights here

			updateConfirmedCabOrders(localCabOrders, confirmedCabOrders, localID, TurnOffCabLightChan, TurnOnCabLightChan)
			ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			CabOrdersToNetworkChan <- localCabOrders

		case a := <-PeersListUpdateCabChan:
			peersList = uniqueIDSlice(a)

		case a := <-LostNodeChan:

			//Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr {
					//If previous state was Inactive, change to Unknown
					if reqArr[floor].State == datatypes.Inactive {
						localCabOrders[a][floor].State = datatypes.Unknown
					}
				}
				CabOrdersToNetworkChan <- localCabOrders

			}

		case a := <-RemoteCabOrdersChan:
			remoteCabOrders := a

			newConfirmedOrInactiveFlag := false

			for remoteID, _ := range remoteCabOrders {
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

					newConfirmedOrInactiveFlag, _ = merge(pLocal, remote, localID, peersList)
				}

			}

			if newConfirmedOrInactiveFlag {
				updateConfirmedCabOrders(localCabOrders, confirmedCabOrders, localID, TurnOffCabLightChan, TurnOnCabLightChan)
				ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			}

			CabOrdersToNetworkChan <- localCabOrders
		}

	}
}
