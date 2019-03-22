package nodeStatesHandler

import (
	"../fsm"
)

type NodeID int

type NodeStateMsg struct {
	ID NodeID
	State fsm.NodeState
}

type NodeStatesHandlerChannels struct {
	LocalNodeStateChan chan fsm.NodeState
	AllNodeStatesChan chan map[NodeID]fsm.NodeState
	NodeLostChan chan NodeID
}

func NodeStatesHandler(
	localID NodeID,
	LocalNodeStateFsmChan <-chan fsm.NodeState,
	AllNodeStatesChan chan<- map[NodeID]fsm.NodeState,
	NodeLost <-chan NodeID,
	BroadcastLocalNodeStateChan chan<- fsm.NodeState,
	RemoteNodeStatesChan <-chan NodeStateMsg) {
	
	var allNodeStates = make(map[NodeID]fsm.NodeState)

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
