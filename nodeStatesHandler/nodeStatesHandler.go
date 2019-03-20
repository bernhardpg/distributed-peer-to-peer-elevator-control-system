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

func NodeStatesHandler(localID fsm.NodeID, LocalNodeStateChan <-chan fsm.NodeState, RemoteNodeStatesChan <-chan fsm.NodeState, 
	AllNodeStatesChan chan<- map[fsm.NodeID]fsm.NodeState) {

//	LocalStateToNetwork := make(chan NodeState)

	var allNodeStates = make(map[fsm.NodeID]fsm.NodeState)

	// TODO remove lost peers from allStates

	for {
		select {

		case a := <-LocalNodeStateChan:
			allNodeStates[localID] = a
//			LocalStateToNetwork <- allNodeStates[localID]
			AllNodeStatesChan <- allNodeStates

		case a := <-RemoteNodeStatesChan:
			allNodeStates[a.ID] = a
			AllNodeStatesChan <- allNodeStates

			fmt.Println(a)
		}

	}

}
