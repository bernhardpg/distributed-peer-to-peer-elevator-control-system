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

func NodeStatesHandler(localID fsm.NodeID, LocalNodeStateChanChan <-chan fsm.NodeState, RemoteNodeStatesChanChan <-chan fsm.NodeState, 
	AllNodeStatesChanChan chan<- map[fsm.NodeID]fsm.NodeState) {

//	LocalStateToNetwork := make(chan NodeState)

	var allNodeStates = make(map[fsm.NodeID]fsm.NodeState)

	// TODO remove lost peers from allStates

	for {
		select {

		case a := <-LocalNodeStateChanChan:
			allNodeStates[localID] = a
//			LocalStateToNetwork <- allNodeStates[localID]
			AllNodeStatesChanChan <- allNodeStates

		case a := <-RemoteNodeStatesChanChan:
			allNodeStates[a.ID] = a
			AllNodeStatesChanChan <- allNodeStates

			fmt.Println(a)
		}

	}

}
