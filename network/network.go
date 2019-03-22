package network

import (
	"time"
	"fmt"
	"strconv"
	"../datatypes"
	"./driver/bcast"
	"./driver/peers"
	"../fsm"
	"../nodeStatesHandler"
	"../consensusModule/hallConsensus"
)


const sendRate = 20 * time.Millisecond

type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
	RemoteNodeStatesChan chan nodeStatesHandler.NodeStateMsg
	LocalHallOrdersChan chan [][] datatypes.Req
}

func Module(
	localID datatypes.NodeID,
	LocalNodeStateChan <-chan fsm.NodeState,
	RemoteNodeStatesChan chan<- nodeStatesHandler.NodeStateMsg,
	NodeLostChan chan<- datatypes.NodeID,
	LocalHallOrdersChan <-chan datatypes.HallOrdersMatrix,
	RemoteHallOrdersChan chan<- datatypes.HallOrdersMatrix,
	PeerlistUpdateHallChan chan<- [] datatypes.NodeID) {

	// Configure Peer List
	// -----
	peerUpdateChan := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool) // Used to signal that the node is unavailable
	go peers.Transmitter(15519, strconv.Itoa(int(localID)), peerTxEnable)
	go peers.Receiver(15519, peerUpdateChan)

	// Setup channels and modules for sending and receiving nodeStatesHandler.NodeStateMsg
	// -----
	localStateTx := make(chan nodeStatesHandler.NodeStateMsg)
	remoteStateRx := make(chan nodeStatesHandler.NodeStateMsg, 10) // Does the buffer need to be this high?
	go bcast.Transmitter(15510, localStateTx)
	go bcast.Receiver(15510, remoteStateRx)

	// Setup channels and modules for sending and receiving localHallOrder matrices
	// -----
	localHallOrdersTx := make(chan hallConsensus.LocalHallOrdersMsg)
	remoteHallOrdersRx := make(chan hallConsensus.LocalHallOrdersMsg)
	go bcast.Transmitter(15511, localHallOrdersTx)
	go bcast.Receiver(15511, remoteHallOrdersRx)

	// Initialize variables
	// -----
	bcastPeriod := 200 * time.Millisecond // TODO change this
	bcastTimer := time.NewTimer(bcastPeriod)

	localState := fsm.NodeState {}
	var localHallOrders datatypes.HallOrdersMatrix
	
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
			if len(a.Lost) != 0 {
				for _, currIDstr := range a.Lost {
					currID,_ := strconv.Atoi(currIDstr)
					NodeLostChan <- (datatypes.NodeID)(currID)
				}
			}

			// Notify consensus modules of changes
			if len(a.Lost) != 0 || len(a.New) != 0 {
				var peerlist [] datatypes.NodeID

				for _, currIDstr := range a.Peers {
					currID,_ := strconv.Atoi(currIDstr)
					peerlist = append(peerlist, (datatypes.NodeID)(currID))
				}

				PeerlistUpdateHallChan <- peerlist
				// TODO cabOrders here
			}
			

		// Transmit local state
		case a := <-LocalNodeStateChan:
			localState = a

		// TODO create channel for NodeLostChan for consensus module

		// Receive remote node states
		case a := <- remoteStateRx:
			// Send all (including local) remoteNodeStates to nodeStatesHandler
			// TODO fix comment: Needs localState as well in case we drop out of network and lose ourself
			RemoteNodeStatesChan <- a

		case a := <- LocalHallOrdersChan:
			localHallOrders = a

		case a := <- remoteHallOrdersRx:
			RemoteHallOrdersChan <- a.HallOrders

		case <-bcastTimer.C:
			bcastTimer.Reset(bcastPeriod)

			localStateTx <- nodeStatesHandler.NodeStateMsg {
				ID: localID,
				State: localState,
			}

			localHallOrdersTx <- hallConsensus.LocalHallOrdersMsg {
				ID: localID, // This is actually never used, but is included for consistency on network
				HallOrders: localHallOrders,
			}
		}
	}
}