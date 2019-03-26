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

	localNodeState := fsm.NodeState{}
	var localHallOrders datatypes.HallOrdersMatrix
	var localCabOrders datatypes.CabOrdersMap

	fmt.Println("(network) Initialized")

	fmt.Printf("(network) peerlist: %v\n", peerlist)
	
	// Handle network traffic
	// -----

	for {
		select {

		case a := <-peerUpdateChan:
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
			// (will get removed from a.Peers when there is no network connection)
			if !consensus.ContainsID(peerlist, localID){
				peerlist = append(peerlist, localID)
			}

			PeerlistUpdateHallChan <- peerlist
			PeerlistUpdateCabChan <- peerlist
			PeerlistUpdateAssignerChan <- peerlist


		// Let FSM toggle network visibility (due to obstructions)
		case a := <- FsmToggleNetworkVisibilityChan:
			peerTxEnable <- a

		// Transmit local state
		case a := <-LocalNodeStateChan:
			localNodeState = a

		// Receive remote node states
		case a := <-remoteStateRx:
			// Send all remoteNodeStates to nodestates, including the one with the localID
			RemoteNodeStatesChan <- a

		case a := <-LocalHallOrdersChan:
			localHallOrders = a

		case a := <-remoteHallOrdersRx:
			// Send all remoteOrders to consensus module, including the one with the localID
			// (Orders can only be confirmed by comparing local and remote cab orders information) 
			RemoteHallOrdersChan <- a.HallOrders

		case a := <-LocalCabOrdersChan:
			localCabOrders = a

		case a := <-remoteCabOrdersRx:
			// Send all remoteOrders to consensus module, including the one with the localID
			// (Orders can only be confirmed by comparing local and remote cab orders information) 
			RemoteCabOrdersChan <- a.CabOrders

		// Broadcast periodically
		case <-bcastTimer.C:
			bcastTimer.Reset(bcastPeriod)

			// Initialize messages to send on network
			// ------
			localNodeStateMsg := nodestates.NodeStateMsg {
				ID:    localID,
				State: localNodeState,
			}

			localCabOrdersMsg := consensus.LocalCabOrdersMsg {
				// This ID is actually never used, but is included for consistency on network
				ID:         localID,
				CabOrders:  localCabOrders,
			}

			localHallOrdersMsg := consensus.LocalHallOrdersMsg {
				// This ID is actually never used, but is included for consistency on network
				ID:         localID,
				HallOrders: localHallOrders,
			}

			// Send localCabOrders and localNodeState directly to remote channels if the node is
			// alone in peerlist.
			// (Orders can only be confirmed by comparing local and remote cab orders information,
			// and nodeStates are only updated when received remotely)
			if consensus.ContainsID(peerlist, localID) && len(peerlist) == 1 {
				RemoteCabOrdersChan <- localCabOrders
				RemoteNodeStatesChan <- localNodeStateMsg
				// (Hall orders is not sent because they won't be accepted when there are no other nodes on the network)
				break
			}

			// Broadcast information if there are other nodes on the network
			// --------
			localStateTx <- localNodeStateMsg
			localHallOrdersTx <- localHallOrdersMsg
			localCabOrdersTx <- localCabOrdersMsg

		}
	}
}
