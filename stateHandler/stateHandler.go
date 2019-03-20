package stateHandler

import ("fmt"
	
)

type StateHandlerChannels struct {
	LocalElevState chan ElevState
	RemoteElevState chan ElevState
	AllElevStates chan map[NodeID] ElevState
}

type BehaviourState int;
const (
	InitState BehaviourState = iota;
	Idle
	DoorOpen
	Moving
)

type OrderDir int;
const (
	Up OrderDir = iota;
	Down;
)

type ElevState struct {
	ID NodeID
	State BehaviourState
	Floor int
	Dir   OrderDir
}

// TODO move to network module!
type NodeID int;

func StateHandler(localID NodeID, LocalElevStateChan <-chan ElevState, RemoteElevStateChan <-chan ElevState, 
	AllElevStatesChan chan<- map[NodeID]ElevState) {

//	LocalStateToNetwork := make(chan ElevState)

	var allElevStates = make(map[NodeID]ElevState)

	// TODO remove lost peers from allStates

	for {
		select {

		case a := <-LocalElevStateChan:
			allElevStates[localID] = a
//			LocalStateToNetwork <- allElevStates[localID]
			AllElevStatesChan <- allElevStates

		case a := <-RemoteElevStateChan:
			allElevStates[a.ID] = a
			AllElevStatesChan <- allElevStates

			fmt.Println(a)
		}

	}

}
