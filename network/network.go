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
	RemoteNodeStatesChan chan<- NodeStateMsg,
	NodeLostChan chan<- NodeID) {

	// Setup channels and modules for sending and receiving NodeStateMsg
	// -----
	localStateTx := make(chan NodeStateMsg)
	remoteStateRx := make(chan NodeStateMsg, 10) // Does the buffer need to be this high?
	go bcast.Transmitter(16569, localStateTx)
	go bcast.Receiver(16569, remoteStateRx)

	// Configure Peer List
	// -----
	peerUpdateChan := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool) // Used to signal that the node is unavailable
	go peers.Transmitter(15647, strconv.Itoa(int(localID)), peerTxEnable)
	go peers.Receiver(15647, peerUpdateChan)


	// Handle network traffic
	// -----

	localState := fsm.NodeState {}
	bcastPeriod := 200 * time.Millisecond
	bcastTimer := time.NewTimer(bcastPeriod)

	for {
		select {

		case a := <-peerUpdateChan:
			// Print info
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", a.Peers)
			fmt.Printf("  New:      %q\n", a.New)
			fmt.Printf("  Lost:     %q\n", a.Lost)

			// Inform NodeStatesHandler that one ore more nodes are lost from the network
			if len(a.Lost) != 0 {
				for _, currIDstr := range a.Lost {
					currID,_ := strconv.Atoi(currIDstr)
					NodeLostChan <- NodeID(currID)
				}
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

		case <-bcastTimer.C:
			bcastTimer.Reset(bcastPeriod)

			localStateTx <- NodeStateMsg {
				ID: localID,
				State: localState,
			}
			
		}
	}
}