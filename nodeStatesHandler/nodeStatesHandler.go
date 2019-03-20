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
}

func NodeStatesHandler(
	localID network.NodeID,
	LocalNodeStateFsmChan <-chan fsm.NodeState,
	RemoteNodeStatesChan <-chan network.NodeStateMsg,
	AllNodeStatesChan chan<- map[network.NodeID]fsm.NodeState,
	BroadcastLocalNodeStateChan chan<- fsm.NodeState) {
	
	var allNodeStates = make(map[network.NodeID]fsm.NodeState)

	// TODO remove lost peers from allStates

	for {
		select {

		case a := <-LocalNodeStateFsmChan:
			allNodeStates[localID] = a

			BroadcastLocalNodeStateChan <- a
			AllNodeStatesChan <- allNodeStates

		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a.State
			AllNodeStatesChan <- allNodeStates
		}
	}
}
