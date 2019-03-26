package nodestates

import (
	"../datatypes"
)

// NodeStateMsg ...
// Used for broadcasting the local node state and for receiving remote node states
type NodeStateMsg struct {
	ID    datatypes.NodeID
	State datatypes.NodeState
}

// Channels ...
// Used for communication between this module and other modules
type Channels struct {
	LocalNodeStateChan chan datatypes.NodeState
	AllNodeStatesChan  chan datatypes.AllNodeStatesMap
	NodeLostChan       chan datatypes.NodeID
}

// deepcopyNodeStates ...
// @return: A pointer to a deep copied map of allNodeStates
func deepcopyNodeStates(m datatypes.AllNodeStatesMap) datatypes.AllNodeStatesMap {
	cpy := make(datatypes.AllNodeStatesMap)

	for currID := range m {
		temp := datatypes.NodeState{
			Behaviour: m[currID].Behaviour,
			Floor:     m[currID].Floor,
			Dir:       m[currID].Dir,
		}
		cpy[currID] = temp
	}

	return cpy
}

// Handler ...
// The nodestates handler keeps an updated state on all nodes currently in the system
// (that is, nodes that are in peerlist).
// Lost nodes will be deleted from the collection of states, and new nodes will
// be added to the collection of states immediately.
func Handler(
	localID datatypes.NodeID,
	FsmLocalNodeStateChan <-chan datatypes.NodeState,
	NetworkAllNodeStatesChan chan<- datatypes.AllNodeStatesMap,
	NodeLost <-chan datatypes.NodeID,
	NetworkLocalNodeStateChan chan<- datatypes.NodeState,
	RemoteNodeStatesChan <-chan NodeStateMsg) {

	var allNodeStates = make(datatypes.AllNodeStatesMap)

	for {
		select {

		// Send received localState from FSM to the network module
		case a := <-FsmLocalNodeStateChan:
			NetworkLocalNodeStateChan <- a

		// Update allNodeStates with the received node state, and
		// update the network module
		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a.State
			NetworkAllNodeStatesChan <- deepcopyNodeStates(allNodeStates)

		// Remove lost nodes from allNodeStates
		case a := <-NodeLost:
			delete(allNodeStates, a)
		}

	}
}
