package StateHandler

import (
	"./fsm"
	//"./netWork"
	"./optimalAssigner"
	"fmt"
)

type ElevStateObject struct {
	state elevState
	floor int
	dir   orderDir
}

type nodeID int

func stateHandler(LocalElevStateChan chan<- ElevStateObject, RemoteElevStatesChan <-chan []ElevStateObject) {

	//do something fun

	//Declare all states list
	//declare local state

	//Make in main?

	var localID nodeID = 1
	LocalStateToNetwork := make(chan ElevStateObject)
	//AllStatesToAssigner := make(chan []ElevStateObject)

	var localState ElevStateObject
	var allStates = make(map[nodeID]ElevStateObject)

	for {
		select {

		case localState := <-LocalElevStateChan:
			if allStates.nodeID == localID {
				allStates[nodeID] = localState
			}
			LocalStateToNetwork <- localState

		case a := <-RemoteElevStatesChan:
			//update remote states (in StateHandler)
		}

	}

}
