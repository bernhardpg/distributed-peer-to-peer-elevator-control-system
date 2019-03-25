package consensus

// ----------
// Cab Order Consensus module
// ----------

import (
	"../datatypes"
	"../elevio"
	"fmt"
)

type CabOrderChannels struct {
	NewOrderChan       chan int
	CompletedOrderChan chan int
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
	numFloors int,
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeerlistUpdateCabChan <-chan []datatypes.NodeID,
	LostNodeChan <-chan datatypes.NodeID,
	RemoteCabOrdersChan <-chan datatypes.CabOrdersMap,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	ConfirmedCabOrdersToAssignerChan chan<- datatypes.ConfirmedCabOrdersMap,
	CabOrdersToNetworkChan chan<- datatypes.CabOrdersMap) {

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
	localCabOrders[localID] = make(datatypes.CabOrdersList, numFloors)

	confirmedCabOrders := calcConfirmedOrders(localCabOrders)

	// Initialize all orders to unknown
	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req{
			State: datatypes.Unknown,
			AckBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}

	// TODO send initialized variables to other modules

	fmt.Println("(consensus) CabOrdersModule initialized")

	// Logic for handling consensus when new data enters system
	// -----
	for {
		select {

		// Store new local orders as pendingAck and update network module
		case a := <-NewCabOrderChan:

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: []datatypes.NodeID{localID},
			}

			CabOrdersToNetworkChan <- deepcopyCabOrders(localCabOrders)

		// Mark completed orders as inactive and update network module and optimalAssigner
		// with all confirmedCabOrders
		case a := <-CompletedCabOrderChan:

			localCabOrders[localID][a] = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}

			confirmedCabOrders = calcConfirmedOrders(localCabOrders)

			// Send updates to optimalAssigner
			// TODO does these need to be deep copied?
			ConfirmedCabOrdersToAssignerChan <- deepcopyConfirmedCabOrders(confirmedCabOrders)

			// Send updates to network module
			CabOrdersToNetworkChan <- deepcopyCabOrders(localCabOrders)

		// Update peerlistpeerlist  with changes received from network module
		case a := <-PeerlistUpdateCabChan:
			peerlist = uniqueIDSlice(a)

		// TODO implement this channel
		// Set all Inactive orders of a lost node to Unknown
		// (To avoid overiding any changes in the nodes' cab orders while offline)
		case a := <-LostNodeChan:

			// Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr {
					if reqArr[floor].State == datatypes.Inactive {
						localCabOrders[a][floor].State = datatypes.Unknown
					}
				}
				CabOrdersToNetworkChan <- localCabOrders

			}

		// Merge received remoteCabOrders from network module with local data in localCabOrders
		case a := <-RemoteCabOrdersChan:

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

					if newInactiveFlag {
						clearCabLight(floor, TurnOffCabLightChan)
					} else if newConfirmedFlag {
						setCabLight(floor, TurnOnCabLightChan)
					}
				}

			}

			// Only update confirmedCabOrders when orders are changed to inactive or confirmed
			if confirmedOrdersChangedFlag {
				calcConfirmedOrders(localCabOrders)
				ConfirmedCabOrdersToAssignerChan <- confirmedCabOrders
			}

			// Update network module with new data
			CabOrdersToNetworkChan <- localCabOrders
		}

	}
}
