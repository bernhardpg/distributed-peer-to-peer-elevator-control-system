package nodeStatesHandler

import (
	"../datatypes"
	"../fsm"
)


type NodeStateMsg struct {
	ID datatypes.NodeID
	State fsm.NodeState
}

type NodeStatesHandlerChannels struct {
	LocalNodeStateChan chan fsm.NodeState
	AllNodeStatesChan chan map[datatypes.NodeID]fsm.NodeState
	NodeLostChan chan datatypes.NodeID
}

func NodeStatesHandler(
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
			//allNodeStates[localID] = a // TODO remove this? Will already be broadcastet from network
			BroadcastLocalNodeStateChan <- a

			//AllNodeStatesChan <- allNodeStates // TODO remove this?

		// Received remoteNodeState
		case a := <-RemoteNodeStatesChan:
//			fmt.Println("(nodeStatesHandler) Updating node: ", a.ID, " in allNodeStates")
			allNodeStates[a.ID] = a.State
			AllNodeStatesChan <- allNodeStates

//			fmt.Println(allNodeStates)
		
		// Remove lost nodes from allNodeStates
		case a := <-NodeLost:
			delete(allNodeStates, a)
//			fmt.Println("(nodeStatesHandler) Removing node: ", a, " from network")
//			fmt.Println(allNodeStates)
		}

	}
}
