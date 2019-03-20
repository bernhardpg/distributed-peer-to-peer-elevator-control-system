package optimalOrderAssigner

import (
	"os/exec"
	"log"
	"os"
	"encoding/json"
	"../fsm"
	"reflect"
	"../elevio"
)

type OptimalOrderAssignerChannels struct {
	HallOrdersChan chan [][] bool
	CabOrdersChan chan [] bool
	NewOrderChan chan elevio.ButtonEvent // TODO move to consensus module
	LocallyAssignedOrdersChan chan [][] bool
	CompletedOrderChan chan int
}

type singleNodeStateJSON struct {
	Behaviour string 	`json:"behaviour"`
	Floor int 			`json:"floor"`
	Direction string 	`json:"direction"`
	CabRequests []bool 	`json:"cabRequests"`
}

type optimizationInputJSON struct {
	// Following format req to use HallRequestAssigner (written in D)
	HallRequests [][]bool 					`json:"hallRequests"`
	States map[string] singleNodeStateJSON 	`json:"states"`
}

// Encodes the data for HallRequstAssigner script, according to
// Format required
func encodeJSON(
	currHallOrdersChan [][]bool,
	currCabOrdersChan []bool,
	currAllNodeStatesChan map[fsm.NodeID] fsm.NodeState)([]byte) {

	currStates := make(map[string] singleNodeStateJSON);

	for currID, currNodeState := range currAllNodeStatesChan {
		currBehaviour := "";
		currDirection := "";

		switch(currNodeState.Behaviour) {

			case fsm.IdleState, fsm.InitState:
				currBehaviour = "idle"
				currDirection = "stop"

			case fsm.MovingState:
				currBehaviour = "moving"

				switch(currNodeState.Dir) {
					case fsm.Up:
						currDirection = "up"
					case fsm.Down:
						currDirection = "down"
				}

			case fsm.DoorOpenState:
				currBehaviour = "doorOpen"
				currDirection = "stop"
		}

		currStates[string(currID)] = singleNodeStateJSON {
			Behaviour: currBehaviour,
			Floor: currNodeState.Floor,
			Direction: currDirection,
			CabRequests: currCabOrdersChan,
		}
	}


	currOptimizationInput := optimizationInputJSON {
		HallRequests: currHallOrdersChan,
		States: currStates,
	}


	currOptimizationInputJSON,_ := json.Marshal(currOptimizationInput);
	return currOptimizationInputJSON;
}

func runOptimizer(currOptimizationInputJSON []byte) ([]byte){
	// Get current working directory
	dir, err := os.Getwd();

	if err != nil {
		log.Fatal(err);
	}

	scriptName := "/optimalOrderAssigner/hall_request_assigner"
	params := "--includeCab --clearRequestType all"
	input := " --input '" + string(currOptimizationInputJSON) + "'"

	// Run external script with json data
	cmd := exec.Command("sh", "-c",
		dir + scriptName + " " + params + " " + input);
	
	outJSON, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	return outJSON;
}

func setOrder(
	buttonPress elevio.ButtonEvent,
	hallOrders [][]bool,
	cabOrders []bool,
	TurnOnLightsChan chan<- elevio.ButtonEvent) {
	if buttonPress.Button == elevio.BT_Cab {
		cabOrders[buttonPress.Floor] = true
	} else {
		hallOrders[buttonPress.Floor][buttonPress.Button] = true;
	}

	TurnOnLightsChan <- buttonPress
}

func clearOrdersAtFloor(
	floor int,
	hallOrders [][]bool,
	cabOrders []bool,
	TurnOffLightsChan chan<- elevio.ButtonEvent) {
	cabOrders[floor] = false
	hallOrders[floor] = []bool{false, false}

	// Clear all buttons
	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
		TurnOffLightsChan <- elevio.ButtonEvent {
			Floor: floor,
			Button: orderType,
		}
	}

}

func Assigner(
	localID fsm.NodeID,
	numFloors int,
	HallOrdersChanChan <-chan [][] bool,
	CabOrdersChanChan <-chan [] bool,
	LocallyAssignedOrdersChan chan<- [][]bool,
	NewOrderChan <-chan elevio.ButtonEvent,
	CompletedOrderChan <-chan int,
	AllNodeStatesChan <-chan map[fsm.NodeID] fsm.NodeState,
	TurnOffLightsChan chan<- elevio.ButtonEvent,
	TurnOnLightsChan chan<- elevio.ButtonEvent) { 

	// Initialize empty matrices
	//-------
	currHallOrdersChan := make([][] bool, numFloors); 
	currCabOrdersChan := make([] bool, numFloors);

	for floor := range currHallOrdersChan {
		currHallOrdersChan[floor] = make([] bool, 2)
	}

	for floor := range currHallOrdersChan {
		for orderType := range currHallOrdersChan[floor] {
			currHallOrdersChan[floor][orderType] = false
		}
		currCabOrdersChan[floor] = false
	}

	currAllNodeStatesChan := make(map[fsm.NodeID] fsm.NodeState);
	var currOptimizationInputJSON []byte;
	var optimalAssignedOrders map[string] [][]bool;
	var prevLocallyAssignedOrders [][]bool; 

	for {
		select {
			case a := <- AllNodeStatesChan:
				currAllNodeStatesChan = a

			case a := <- NewOrderChan:
				setOrder(a, currHallOrdersChan, currCabOrdersChan, TurnOnLightsChan)

			case a := <- CompletedOrderChan:
				clearOrdersAtFloor(a, currHallOrdersChan, currCabOrdersChan, TurnOffLightsChan)
		}

		// Calculate new optimalAssignedOrders time a message is received
		currOptimizationInputJSON = encodeJSON(currHallOrdersChan, currCabOrdersChan, currAllNodeStatesChan);
		outJSON := runOptimizer(currOptimizationInputJSON);
		json.Unmarshal(outJSON, &optimalAssignedOrders);

		currLocallyAssignedOrders := optimalAssignedOrders[string(localID)]

		// No changes, don't send updated orders
		if reflect.DeepEqual(currLocallyAssignedOrders, prevLocallyAssignedOrders) {
			continue;
		}

		LocallyAssignedOrdersChan <- currLocallyAssignedOrders
		prevLocallyAssignedOrders = currLocallyAssignedOrders
	}
}