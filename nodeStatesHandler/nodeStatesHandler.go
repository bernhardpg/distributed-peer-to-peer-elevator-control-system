package nodeStatesHandler

import (
	"fmt"
	"../fsm"
)

type NodeStatesHandlerChannels struct {
	LocalNodeStateChan chan fsm.NodeState
	RemoteNodeStatesChan chan fsm.NodeState
	AllNodeStatesChan chan map[fsm.NodeID]fsm.NodeState
}

func NodeStatesHandler(
	localID fsm.NodeID,
	LocalNodeStateFsmChan <-chan fsm.NodeState,
	RemoteNodeStatesChan <-chan fsm.NodeState, 
	AllNodeStatesChan chan<- map[fsm.NodeID]fsm.NodeState,
	BroadcastLocalNodeStateChan chan<- fsm.NodeState) {
	
	var allNodeStates = make(map[fsm.NodeID]fsm.NodeState)

	// TODO remove lost peers from allStates

	for {
		select {

		case a := <-LocalNodeStateFsmChan:
			allNodeStates[localID] = a

			BroadcastLocalNodeStateChan <- allNodeStates[localID]
			AllNodeStatesChan <- allNodeStates

		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a
			AllNodeStatesChan <- allNodeStates

			fmt.Println(a)
		}

	}

}
