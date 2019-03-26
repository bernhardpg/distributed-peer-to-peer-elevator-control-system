package consensus

import (
	"../datatypes"
	"../elevio"
	"fmt"
	//"github.com/jinzhu/copier"
)

// CabOrderChannels ...
// Channels used for communication related to consensus of cab orders with other modules
type CabOrderChannels struct {
	NewOrderChan        chan int
	CompletedOrderChan  chan int
	ConfirmedOrdersChan chan datatypes.ConfirmedCabOrdersMap
	LocalOrdersChan     chan datatypes.CabOrdersMap
	RemoteOrdersChan    chan datatypes.CabOrdersMap
	PeerlistUpdateChan  chan []datatypes.NodeID
	LostPeerChan        chan datatypes.NodeID
}

// LocalCabOrdersMsg ...
// Used for broadcasting localHallOrders to other nodes
type LocalCabOrdersMsg struct {
	ID        datatypes.NodeID
	CabOrders datatypes.CabOrdersMap
}

// calcConfirmedCabOrders ...
// @return: map with boolean arrays where only Confirmed orders are set to true
func calcConfirmedOrders(localCabOrders datatypes.CabOrdersMap) datatypes.ConfirmedCabOrdersMap {

	confirmedOrders := make(datatypes.ConfirmedCabOrdersMap)

	for currID := range localCabOrders {
		confirmedOrders[currID] = make(datatypes.ConfirmedCabOrdersList, len(localCabOrders[currID]))

		for floor := range localCabOrders[currID] {
			if localCabOrders[currID][floor].State == datatypes.Confirmed {
				confirmedOrders[currID][floor] = true
			} else {
				confirmedOrders[currID][floor] = false
			}
		}
	}

	return confirmedOrders
}

// deepcopyCabOrders ...
// @return: A pointer to a hard copied map of type CabOrdersMap
func deepcopyCabOrders(m datatypes.CabOrdersMap) datatypes.CabOrdersMap {
	cpy := make(datatypes.CabOrdersMap)

	for currID, currCabOrderList := range m {
		tempCabOrdersList := make(datatypes.CabOrdersList, len(currCabOrderList))

		for currReqIndex, currReq := range currCabOrderList {

			currAckBy := currReq.AckBy
			tempAckBy := make([]datatypes.NodeID, len(currAckBy))
			copy(tempAckBy, currAckBy)

			tempReq := datatypes.Req{
				State: currReq.State,
				AckBy: currReq.AckBy,
			}

			tempCabOrdersList[currReqIndex] = tempReq
		}

		cpy[currID] = tempCabOrdersList
	}

	return cpy
}

// deepcopyConfirmedCabOrders ...
// @return: A pointer to a hard copied map of type ConfirmedCabOrdersMap
func deepcopyConfirmedCabOrders(m datatypes.ConfirmedCabOrdersMap) datatypes.ConfirmedCabOrdersMap {
	cpy := make(datatypes.ConfirmedCabOrdersMap)

	for currID := range m {
		temp := make(datatypes.ConfirmedCabOrdersList, len(m[currID]))
		copy(temp, m[currID])
		cpy[currID] = temp
	}

	return cpy
}

// CabOrdersModule ...
// Handles the information distribution for cab orders between nodes.
// Keeps track of which orders are currently confirmed by all nodes, which orders are still
// pending acknowledgement, and which orders are completed (Inactive). Only
// confirmed orders are passed along to the optimal assigner, making sure
// that all nodes agree on the distribution of all of the orders at all times.
func CabOrdersModule(
	localID datatypes.NodeID,
	NewOrderChan <-chan int,
	ConfirmedOrdersChan chan<- datatypes.ConfirmedCabOrdersMap,
	CompletedOrderChan <-chan int,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	LocalOrdersChan chan<- datatypes.CabOrdersMap,
	RemoteOrdersChan <-chan datatypes.CabOrdersMap,
	PeerlistUpdateChan <-chan []datatypes.NodeID,
	LostPeerChan <-chan datatypes.NodeID) {

	// Initialize variables
	// ----
	peerlist := []datatypes.NodeID{}

	// Note: These variables are initialized dynamically, as opposed to the hall order matrices.
	// Hence these values will need to be deep copied before being sent on any channels, as the
	// values sent on the channels are merely pointers.
	// (This is because of the way Golang handles memory allocation for maps,
	// where maps are always passed by reference and where arrays within maps need to be initialized
	// dynamically in order to get assigned new values later)
	localCabOrders := make(datatypes.CabOrdersMap)
	localCabOrders[localID] = make(datatypes.CabOrdersList, elevio.NumFloors)

	// Initialize all orders to unknown to allow inheritance of data from the network.
	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req{
			State: datatypes.Unknown,
			AckBy: nil,
		}
	}

	// Send initialized variables to orderassigner and network module
	confirmedCabOrders := calcConfirmedOrders(localCabOrders)
	ConfirmedOrdersChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)
	LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

	fmt.Println("(consensus:caborders) Initialized")

	// Logic for handling consensus when new data enters system
	// -----
	for {
		select {

		// Store new local orders as pendingAck and update network module
		case a := <-NewOrderChan:

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: []datatypes.NodeID{localID},
			}

			// Send updates to network module
			LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

		// Mark completed orders (with localID) as inactive and update network
		// module and optimalAssigner with all confirmedCabOrders
		case a := <-CompletedOrderChan:
			// (Will only clear cab order with own ID, as only locally completed
			// orders are passed on this channel)
			clearCabLight(a, TurnOffCabLightChan)

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}

			confirmedCabOrders = calcConfirmedOrders(localCabOrders)
			// Send updates to optimalAssigner
			ConfirmedOrdersChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)
			// Send updates to network module
			LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

		// Update peerlist with changes received from network module
		case a := <-PeerlistUpdateChan:
			peerlist = UniqueIDSlice(a)

		// Set all Inactive orders of a lost node to Unknown and update network module
		// (To avoid overiding any changes in the nodes' cab orders while offline)
		case a := <-LostPeerChan:

			// Only set orders to Unknown if peer already was in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr {
					if reqArr[floor].State == datatypes.Inactive {
						localCabOrders[a][floor].State = datatypes.Unknown
					}
				}
				// Send updates to network module
				LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

			}

		// Merge received remoteCabOrders from network module with local data in localCabOrders
		case a := <-RemoteOrdersChan:

			remoteCabOrders := deepcopyCabOrders(a)
			confirmedOrdersChangedFlag := false

			// Merge world views for every order on every node in CabOrder map
			for remoteID := range remoteCabOrders {

				// Always add all data on new nodes to CabOrder map
				_, existsInMap := localCabOrders[remoteID]
				if !existsInMap {
					localCabOrders[remoteID] = remoteCabOrders[remoteID]
					continue
				}

				// Merge every order for each ID in remoteCabOrders
				reqArr := localCabOrders[remoteID]
				for floor := range reqArr {
					pLocal := &localCabOrders[remoteID][floor]
					remote := remoteCabOrders[remoteID][floor]

					newInactiveFlag, newConfirmedFlag := merge(pLocal, remote, localID, peerlist)

					// Make flag stay true if set to true once
					confirmedOrdersChangedFlag = confirmedOrdersChangedFlag || newInactiveFlag || newConfirmedFlag

					// Handle lights if the local order is set to Inactive or Confirmed
					if newInactiveFlag && remoteID == localID {
						clearCabLight(floor, TurnOffCabLightChan)
					} else if newConfirmedFlag && remoteID == localID {
						setCabLight(floor, TurnOnCabLightChan)
					}
				}
			}

			// Only update confirmedCabOrders when orders are changed to Inactive or Confirmed
			if confirmedOrdersChangedFlag {
				confirmedCabOrders = calcConfirmedOrders(localCabOrders)
				ConfirmedOrdersChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)
			}

			// Update network module with new data
			LocalOrdersChan <- deepcopyCabOrders(localCabOrders)
		}
	}
}
