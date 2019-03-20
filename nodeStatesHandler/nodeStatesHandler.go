package nodeStatesHandler

import (
	"fmt"
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
			allNodeStates[localID] = a

			BroadcastLocalNodeStateChan <- a
			AllNodeStatesChan <- allNodeStates

		// Received remoteNodeState
		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a.State
			AllNodeStatesChan <- allNodeStates

			fmt.Println(allNodeStates)
		
		// Remove lost nodes from allNodeStates
		case a := <-NodeLost:
			delete(allNodeStates, a)
			fmt.Println(allNodeStates)
		}

	}
}
