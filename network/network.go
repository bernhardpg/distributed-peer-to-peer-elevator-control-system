package network

import (
	"time"
	"fmt"
	"./driver/bcast"
	"../fsm"
)

const sendRate = 20 * time.Millisecond

type NodeID int;

type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
}

type localStateMsg struct {
	ID NodeID
	State fsm.NodeState
}

func Module(
	localID NodeID,
	LocalNodeStateChan <-chan fsm.NodeState) {

	// Setup channels and modules for sending and receiving localStateMsg
	localStateTx := make(chan localStateMsg)
	localStateRx := make(chan localStateMsg)
	go bcast.Transmitter(16569, localStateTx)
	go bcast.Receiver(16569, localStateRx)

	
	localState := fsm.NodeState {}

	for {
		select {

		case a := <-LocalNodeStateChan:
			localState = a
			localStateTx <- localStateMsg {
				ID: localID,
				State: localState,
			}

		case a := <- localStateRx:
			if a.ID != localID {
				fmt.Println("(network) Received from id: ", a.ID)
				fmt.Println("   State: ", a.State)
			}
		}
	}
}