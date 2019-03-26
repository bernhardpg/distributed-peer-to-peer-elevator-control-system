package network

import (
	"../consensus"
	"../datatypes"
	"../fsm"
	"../nodestates"
	"./driver/bcast"
	"./driver/peers"
	"time"
	"fmt"
)


type Channels struct {
	LocalNodeStateChan   chan fsm.NodeState
	RemoteNodeStatesChan chan nodestates.NodeStateMsg
	LocalHallOrdersChan  chan [][]datatypes.Req
}

func Module(
	localID datatypes.NodeID,
	FsmToggleNetworkVisibilityChan <-chan bool,
	LocalNodeStateChan <-chan fsm.NodeState,
	RemoteNodeStatesChan chan<- nodestates.NodeStateMsg,
	NodeLostChan chan<- datatypes.NodeID,
	PeerlistUpdateAssignerChan chan<- []datatypes.NodeID,
	LocalHallOrdersChan <-chan datatypes.HallOrdersMatrix,
	RemoteHallOrdersChan chan<- datatypes.HallOrdersMatrix,
	PeerlistUpdateHallChan chan<- []datatypes.NodeID,
	LocalCabOrdersChan <-chan datatypes.CabOrdersMap,
	RemoteCabOrdersChan chan<- datatypes.CabOrdersMap,
	PeerlistUpdateCabChan chan<- []datatypes.NodeID,
	LostPeerCabChan chan<- datatypes.NodeID) {

	// Configure Peer List
	// -----
	peerUpdateChan := make(chan peers.PeerUpdate,10)
	peerTxEnable := make(chan bool) // Used to signal that the node is unavailable
	go peers.Transmitter(15519, string(localID), peerTxEnable)
	go peers.Receiver(15519, peerUpdateChan)

	// Setup channels and modules for sending and receiving nodestates.NodeStateMsg
	// -----
	localStateTx := make(chan nodestates.NodeStateMsg)
	remoteStateRx := make(chan nodestates.NodeStateMsg, 10) // Does the buffer need to be this high?
	go bcast.Transmitter(15510, localStateTx)
	go bcast.Receiver(15510, remoteStateRx)

	// Setup channels and modules for sending and receiving localHallOrder matrices
	// -----
	localHallOrdersTx := make(chan consensus.LocalHallOrdersMsg)
	remoteHallOrdersRx := make(chan consensus.LocalHallOrdersMsg, 10)
	go bcast.Transmitter(15511, localHallOrdersTx)
	go bcast.Receiver(15511, remoteHallOrdersRx)

	// Setup channels and modules for sending and receiving localCabOrder maps
	// -----
	localCabOrdersTx := make(chan consensus.LocalCabOrdersMsg)
	remoteCabOrdersRx := make(chan consensus.LocalCabOrdersMsg, 10)
	go bcast.Transmitter(15512, localCabOrdersTx)
	go bcast.Receiver(15512, remoteCabOrdersRx)

	// Initialize variables
	// -----
	peerlist := []datatypes.NodeID{localID}

	bcastPeriod := 50 * time.Millisecond // TODO change this
	bcastTimer := time.NewTimer(bcastPeriod)

	localState := fsm.NodeState{}
	var localHallOrders datatypes.HallOrdersMatrix
	var localCabOrders datatypes.CabOrdersMap

	fmt.Println("(network) Initialized")

	fmt.Printf("(network) peerlist: %v\n", peerlist)
	
	// Handle network traffic
	// -----

	for {
		select {

		case a := <-peerUpdateChan:
			// Print info
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", a.Peers)
			fmt.Printf("  New:      %q\n", a.New)
			fmt.Printf("  Lost:     %q\n", a.Lost)

			// Inform NodeStatesHandler and consensusModules that one ore more nodes are lost from the network
			for _, currID := range a.Lost {
				NodeLostChan <- (datatypes.NodeID)(currID)
				LostPeerCabChan <- (datatypes.NodeID)(currID)
			}

			// Empty peerlist and add all new peers as NodeIDs
			peerlist = []datatypes.NodeID{}
			for _, currID := range a.Peers {
				peerlist = append(peerlist, (datatypes.NodeID)(currID))
			}

			// Make sure that the current node is always in peerlist
			if !consensus.ContainsID(peerlist, localID){
				peerlist = append(peerlist, localID)
			}
			fmt.Printf("(network) peerlist: %v\n", peerlist)

			PeerlistUpdateHallChan <- peerlist
			PeerlistUpdateCabChan <- peerlist
			PeerlistUpdateAssignerChan <- peerlist




		// Let FSM toggle network visibility (due to obstructions)
		case a := <- FsmToggleNetworkVisibilityChan:
			peerTxEnable <- a

		// Transmit local state
		case a := <-LocalNodeStateChan:
			localState = a

		// Receive remote node states
		case a := <-remoteStateRx:
			// Send all (including local) remoteNodeStates to nodestates
			// TODO fix comment: Needs localState as well in case we drop out of network and lose ourself
			RemoteNodeStatesChan <- a

		case a := <-LocalHallOrdersChan:
			localHallOrders = a

		case a := <-remoteHallOrdersRx:
			RemoteHallOrdersChan <- a.HallOrders

		case a := <-LocalCabOrdersChan:
			localCabOrders = a

		case a := <-remoteCabOrdersRx:
			RemoteCabOrdersChan <- a.CabOrders

		case <-bcastTimer.C:
			bcastTimer.Reset(bcastPeriod)

			// Send cabOrders directly as remote if alone in peerlist.
			// (Orders can only be confirmed by comparing local and remote cab orders information) 
			if consensus.ContainsID(peerlist, localID) && len(peerlist) == 1 {
				RemoteCabOrdersChan <- localCabOrders
				break
			}

			localStateTx <- nodestates.NodeStateMsg {
				ID:    localID,
				State: localState,
			}

			localHallOrdersTx <- consensus.LocalHallOrdersMsg {
				ID:         localID, // This is actually never used, but is included for consistency on network
				HallOrders: localHallOrders,
			}

			localCabOrdersTx <- consensus.LocalCabOrdersMsg {
				ID:         localID, // This is actually never used, but is included for consistency on network
				CabOrders:  localCabOrders,
			}

		}
	}
}
