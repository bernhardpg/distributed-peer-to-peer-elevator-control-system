package nodeStatesHandler

import (
//	"fmt"
	"../fsm"
	"../network"
)

type NodeStatesHandlerChannels struct {
	LocalNodeStateChan chan fsm.NodeState
	RemoteNodeStatesChan chan network.NodeStateMsg
	AllNodeStatesChan chan map[network.NodeID]fsm.NodeState
	NodeLostChan chan network.NodeID
}

func NodeStatesHandler(
	localID network.NodeID,
	LocalNodeStateFsmChan <-chan fsm.NodeState,
	RemoteNodeStatesChan <-chan network.NodeStateMsg,
	AllNodeStatesChan chan<- map[network.NodeID]fsm.NodeState,
	NodeLost <-chan network.NodeID,
	BroadcastLocalNodeStateChan chan<- fsm.NodeState) {
	
	var allNodeStates = make(map[network.NodeID]fsm.NodeState)

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
