package stateHandler

import ("fmt"
	

)

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
	State BehaviourState
	Floor int
	Dir   OrderDir
}

type nodeID int;

func stateHandler(LocalElevStateChan <-chan ElevState, RemoteElevStatesChan <-chan []ElevState) {

	//do something fun

	//Declare all states list
	//declare local state

	//Make in main?
	var localID nodeID = 1


	LocalStateToNetwork := make(chan ElevState)
	//AllStatesToAssigner := make(chan []ElevState)

	//var localState ElevState

	var allStates = make(map[nodeID]ElevState)

	for {
		select {

		case localState := <-LocalElevStateChan:
			allStates[localID] = localState
			LocalStateToNetwork <- localState

		case a := <-RemoteElevStatesChan:
			//update remote states (in StateHandler)
			fmt.Println(a)
		}

	}

}
