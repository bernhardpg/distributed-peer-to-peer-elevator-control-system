package nodestates

import (
	"../datatypes"
	"../fsm"
)

// NodeStateMsg ...
// Used for broadcasting the local node state and for receiving remote node states
type NodeStateMsg struct {
	ID    datatypes.NodeID
	State fsm.NodeState
}

// Channels ...
// Used for communication between this module and other modules
type Channels struct {
	LocalNodeStateChan chan fsm.NodeState
	AllNodeStatesChan  chan map[datatypes.NodeID]fsm.NodeState
	NodeLostChan       chan datatypes.NodeID
}

func deepcopyNodeStates(m map[datatypes.NodeID]fsm.NodeState) map[datatypes.NodeID]fsm.NodeState {
	cpy := make(map[datatypes.NodeID]fsm.NodeState)

	for currID := range m {
		temp := fsm.NodeState {
			Behaviour: m[currID].Behaviour,
			Floor: m[currID].Floor,
			Dir: m[currID].Dir,
		}
		cpy[currID] = temp
	}

	return cpy
}

// Handler ...
// Keeps an updated state on all nodes currently on the network (in peerlist).
// Lost nodes will be deleted from the collection of states, and new nodes will
// be added to the collection of states immediately.
func Handler(
	localID datatypes.NodeID,
	LocalNodeStateFsmChan <-chan fsm.NodeState,
	AllNodeStatesChan chan<- map[datatypes.NodeID]fsm.NodeState,
	NodeLost <-chan datatypes.NodeID,
	BroadcastLocalNodeStateChan chan<- fsm.NodeState,
	RemoteNodeStatesChan <-chan NodeStateMsg) {

	var allNodeStates = make(map[datatypes.NodeID]fsm.NodeState)

	for {
		select {

		// Received localState from FSM
		case a := <-LocalNodeStateFsmChan:
			BroadcastLocalNodeStateChan <- a

		// Received remoteNodeState
		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a.State
			AllNodeStatesChan <- deepcopyNodeStates(allNodeStates)

		// Remove lost nodes from allNodeStates
		case a := <-NodeLost:
			delete(allNodeStates, a)
		}

	}
}
