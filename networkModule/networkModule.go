package networkModule

import (
	"time"
	"../fsm"
)

const sendRate = 20 * time.Millisecond

type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
}

type localNodeStateMsg struct {
	ID fsm.NodeID
	State fsm.NodeState
}

func Module(
	LocalNodeStateChan <-chan fsm.NodeState) {
	
	localState := fsm.NodeState {}

	for {
		select {

		case a := <-LocalNodeStateChan:
			localState = a
		}
	}
}