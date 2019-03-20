package network

import (
	"time"
	"fmt"
	"./driver/bcast"
	"./driver/peers"
	"strconv"
	"../fsm"
)

const sendRate = 20 * time.Millisecond

type NodeID int;

type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
}

type NodeStateMsg struct {
	ID NodeID
	State fsm.NodeState
}

func Module(
	localID NodeID,
	LocalNodeStateChan <-chan fsm.NodeState,
	RemoteNodeStatesChan chan<- NodeStateMsg) {

	// Setup channels and modules for sending and receiving NodeStateMsg
	// -----
	localStateTx := make(chan NodeStateMsg)
	localStateRx := make(chan NodeStateMsg)
	go bcast.Transmitter(16569, localStateTx)
	go bcast.Receiver(16569, localStateRx)

	// Configure Peer List
	// -----
	peerUpdateChan := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool) // Used to signal that the node is unavailable
	go peers.Transmitter(15647, strconv.Itoa(int(localID)), peerTxEnable)
	go peers.Receiver(15647, peerUpdateChan)


	// Handle network traffic
	// -----

	localState := fsm.NodeState {}

	for {
		select {

		case a := <-peerUpdateChan:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", a.Peers)
			fmt.Printf("  New:      %q\n", a.New)
			fmt.Printf("  Lost:     %q\n", a.Lost)

		// Transmit local state
		case a := <-LocalNodeStateChan:
			localState = a
			localStateTx <- NodeStateMsg {
				ID: localID,
				State: localState,
			}

		// Receive remote node states
		case a := <- localStateRx:
			// Send remoteNodeStates to nodeStatesHandler
			if a.ID != localID {
				RemoteNodeStatesChan <- a
			}
		}
	}
}