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

type NodeID int;

func StateHandler(LocalElevStateChan <-chan ElevState, RemoteElevStateChan <-chan ElevState, 
	AllElevStatesChan chan<- map[NodeID]ElevState) {

	//do something fun

	//Declare all states list
	//declare local state

	//Make in main?
	var localID NodeID = 1

//	LocalStateToNetwork := make(chan ElevState)
	//AllStatesToAssigner := make(chan []ElevState)

	//var localState ElevState
	var allElevStates = make(map[NodeID]ElevState)


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
