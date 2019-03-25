package consensus

import (
	"../datatypes"
	"../elevio"
	"fmt"
)

// LocalCabOrdersMsg ...
// Used for broadcasting localHallOrders to other nodes
type LocalCabOrdersMsg struct {
	ID         	datatypes.NodeID
	CabOrders 	datatypes.CabOrdersMap
}

type CabOrderChannels struct {
	NewOrderChan        chan int
	CompletedOrderChan  chan int
	ConfirmedOrdersChan chan datatypes.ConfirmedCabOrdersMap
	LocalOrdersChan     chan datatypes.CabOrdersMap
	RemoteOrdersChan    chan datatypes.CabOrdersMap
	PeerlistUpdateChan  chan []datatypes.NodeID
	LostPeerChan  		chan datatypes.NodeID
}
// calcConfirmedCabOrders ...
// Constructs a map with boolean arrays where only Confirmed orders are set to true
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

func deepcopyCabOrders(m datatypes.CabOrdersMap) datatypes.CabOrdersMap {
	cpy := make(datatypes.CabOrdersMap)

	for currID := range m {
		temp := make(datatypes.CabOrdersList, len(m[currID]))
		copy(temp, m[currID])
		cpy[currID] = temp
	}

	return cpy
}

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
// TODO write this
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

	// Initialize all orders to unknown
	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req{
			State: datatypes.Unknown,
			AckBy: nil,
		}
	}

	confirmedCabOrders := calcConfirmedOrders(localCabOrders)

	ConfirmedOrdersChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)
	LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

	// TODO send initialized variables to other modules??

	fmt.Println("(consensus) CabOrdersModule initialized")

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

			LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

		// Mark completed orders (with localID) as inactive and update network module and optimalAssigner
		// with all confirmedCabOrders
		case a := <-CompletedOrderChan:
			// Will only clear on own ID
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

		// Update peerlistpeerlist  with changes received from network module
		case a := <-PeerlistUpdateChan:
			peerlist = uniqueIDSlice(a)

		// Set all Inactive orders of a lost node to Unknown
		// (To avoid overiding any changes in the nodes' cab orders while offline)
		case a := <-LostPeerChan:

			// Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr {
					if reqArr[floor].State == datatypes.Inactive {
						localCabOrders[a][floor].State = datatypes.Unknown
					}
				}
				LocalOrdersChan <- deepcopyCabOrders(localCabOrders)

			}

		// Merge received remoteCabOrders from network module with local data in localCabOrders
		case a := <-RemoteOrdersChan:

			remoteCabOrders := a
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

					if newInactiveFlag && remoteID == localID{
						clearCabLight(floor, TurnOffCabLightChan)
					} else if newConfirmedFlag && remoteID == localID{
						setCabLight(floor, TurnOnCabLightChan)
					}
				}
			}

			
			// Only update confirmedCabOrders when orders are changed to inactive or confirmed
			if confirmedOrdersChangedFlag {
				confirmedCabOrders = calcConfirmedOrders(localCabOrders)
				ConfirmedOrdersChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)
			}

			// Update network module with new data
			LocalOrdersChan <- deepcopyCabOrders(localCabOrders)
		}

	}
}
