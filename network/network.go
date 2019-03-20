package network

import (
	"time"
	"fmt"
	"../fsm"
)

const sendRate = 20 * time.Millisecond

type NodeID int;

type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
}

type localNodeStateMsg struct {
	ID NodeID
	State fsm.NodeState
}

func Module(
	LocalNodeStateChan <-chan fsm.NodeState) {
	
	localState := fsm.NodeState {}

	for {
		select {

		case a := <-LocalNodeStateChan:
			localState = a
			fmt.Println(localState)
		}
	}
}