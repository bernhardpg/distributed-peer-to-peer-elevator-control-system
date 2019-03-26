package network

import (
	"../consensus"
	"../datatypes"
	"../fsm"
	"../nodestates"
	"./driver/bcast"
	"./driver/peers"
	"time"
)


type Channels struct {
	LocalNodeStateChan   chan fsm.NodeState
	RemoteNodeStatesChan chan nodestates.NodeStateMsg
	LocalHallOrdersChan  chan [][]datatypes.Req
}

func Module(
	localID datatypes.NodeID,
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
	bcastPeriod := 200 * time.Millisecond // TODO change this
	bcastTimer := time.NewTimer(bcastPeriod)

	localState := fsm.NodeState{}
	var localHallOrders datatypes.HallOrdersMatrix
	var localCabOrders datatypes.CabOrdersMap

	// Handle network traffic
	// -----

	for {
		select {

		case a := <-peerUpdateChan:
			/*// Print info
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", a.Peers)
			fmt.Printf("  New:      %q\n", a.New)
			fmt.Printf("  Lost:     %q\n", a.Lost)*/

			// Inform NodeStatesHandler and consensusModules that one ore more nodes are lost from the network
			if len(a.Lost) != 0 {
				for _, currID := range a.Lost {
					NodeLostChan <- (datatypes.NodeID)(currID)
					LostPeerCabChan <- (datatypes.NodeID)(currID)
				}
			}

			// Notify consensus modules of changes
			if len(a.Lost) != 0 || len(a.New) != 0 {
				var peerlist []datatypes.NodeID

				for _, currID := range a.Peers {
					peerlist = append(peerlist, (datatypes.NodeID)(currID))
				}

				PeerlistUpdateHallChan <- peerlist
				PeerlistUpdateCabChan <- peerlist
				PeerlistUpdateAssignerChan <- peerlist
			}

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
