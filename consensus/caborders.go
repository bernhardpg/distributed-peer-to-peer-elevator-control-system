package consensus

// ----------
// Cab Order Consensus
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

// updateConfirmedCabOrders ...
// Constructs a map with boolean arrays where only Confirmed orders are set to true
func updateConfirmedCabOrders(
	localCabOrders datatypes.CabOrdersMap,
	confirmedCabOrders datatypes.ConfirmedCabOrdersMap,
	localID datatypes.NodeID,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent) {

	// TODO update new nodes in confirmedCabOrders
	for cabID := range localCabOrders {

		for floor := range localCabOrders[cabID] {
			if localCabOrders[cabID][floor].State == datatypes.Confirmed {
				confirmedCabOrders[cabID][floor] = true

			} else {
				confirmedCabOrders[cabID][floor] = false
			}
		}
	}
}

func CabOrdersModule(
	localID datatypes.NodeID,
	numFloors int,
	NewCabOrderChan <-chan int,
	CompletedCabOrderChan <-chan int,
	PeersListUpdateCabChan <-chan []datatypes.NodeID,
	LostNodeChan <-chan datatypes.NodeID,
	RemoteCabOrdersChan <-chan datatypes.CabOrdersMap,
	TurnOffCabLightChan chan<- elevio.ButtonEvent,
	TurnOnCabLightChan chan<- elevio.ButtonEvent,
	ConfirmedCabOrdersToAssignerChan chan<- datatypes.ConfirmedCabOrdersMap,
	CabOrdersToNetworkChan chan<- datatypes.CabOrdersMap) {

	// Initialize variables
	// ----
	localCabOrders := make(datatypes.CabOrdersMap)
	confirmedCabOrders := make(datatypes.ConfirmedCabOrdersMap)

	// Note: These variables are initialized dynamically, as opposed to the hall order matrices.
	// Hence these values will need to be deep copied before being sent on any channels, as the
	// values sent on the channels are merely pointers.
	// (This is because of the way Golang handles memory allocation for maps)
	localCabOrders[localID] = make(datatypes.CabOrdersList, numFloors)
	confirmedCabOrders[localID] = make(datatypes.ConfirmedCabOrdersList, numFloors)

	// Initialize all orders to unknown
	for floor := range localCabOrders[localID] {
		localCabOrders[localID][floor] = datatypes.Req{
			State: datatypes.Unknown,
			AckBy: nil,
		}
		confirmedCabOrders[localID][floor] = false
	}

	peersList := []datatypes.NodeID{}

	fmt.Println("(consensus) CabOrdersModule initialized")

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

			// Assert node is in localCabOrders
			if reqArr, ok := localCabOrders[a]; ok {
				for floor := range reqArr {
					if reqArr[floor].State == datatypes.Inactive {
						localCabOrders[a][floor].State = datatypes.Unknown
					}
				}
				CabOrdersToNetworkChan <- localCabOrders

			}

		case a := <-RemoteCabOrdersChan:
			remoteCabOrders := a

			newConfirmedOrInactiveFlag := false

			for remoteID := range remoteCabOrders {
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
