package optimalAssigner

import (
	"os/exec"
	"log"
	"os"
	"encoding/json"
	"../fsm"
	"../elevio"
)

type OptimalAssignerChannels struct {
	HallOrdersChan chan [][] bool
	CabOrdersChan chan [] bool
	NewOrderChan chan elevio.ButtonEvent // TODO move to consensus module
	LocallyAssignedOrdersChan chan [][] bool
	CompletedOrderChan chan int
}

type singleNodeStateJson struct {
	Behaviour string 	`json:"behaviour"`
	Floor int 			`json:"floor"`
	Direction string 	`json:"direction"`
	CabRequests []bool 	`json:"cabRequests"`
}

type optimizationInputJson struct {
	// Following format req to use HallRequestAssigner (written in D)
	HallRequests [][]bool 					`json:"hallRequests"`
	States map[string] singleNodeStateJson 	`json:"states"`
}

// Encodes the data for HallRequstAssigner script, according to
// Format required
func encodeJson(
	currHallOrdersChan [][]bool,
	currCabOrdersChan []bool,
	currAllNodeStatesChan map[fsm.NodeID] fsm.NodeState)([]byte) {

	currStates := make(map[string] singleNodeStateJson);

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

		currStates[string(currID)] = singleNodeStateJson {
			Behaviour: currBehaviour,
			Floor: currNodeState.Floor,
			Direction: currDirection,
			CabRequests: currCabOrdersChan,
		}
	}


	currOptimizationInput := optimizationInputJson {
		HallRequests: currHallOrdersChan,
		States: currStates,
	}

	currOptimizationInputJson,_ := json.Marshal(currOptimizationInput);
	return currOptimizationInputJson;
}

func runOptimizer(currOptimizationInputJson []byte) ([]byte){
	// Get current working directory
	dir, err := os.Getwd();

	if err != nil {
		log.Fatal(err);
	}

	scriptName := "/optimalAssigner/hall_request_assigner"
	params := "--includeCab --clearRequestType all"
	input := " --input '" + string(currOptimizationInputJson) + "'"

	// Run external script with json data
	cmd := exec.Command("sh", "-c",
		dir + scriptName + " " + params + " " + input);
	
	outJson, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	return outJson;
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

	// TODO set lights
	//if arrived at ordered floor && !moving:
	//	Turn off all lights on current floor
	//if button pressed:
	//	turn on button pressed order lamp

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
	var currOptimizationInputJson []byte;
	var optimalAssignedOrders map[string] [][]bool;
	calcOptimalFlag := false;

	for {
		select {
			case a := <- AllNodeStatesChan:
				currAllNodeStatesChan = a
				calcOptimalFlag = true

			case a := <- NewOrderChan:
				setOrder(a, currHallOrdersChan, currCabOrdersChan, TurnOnLightsChan)
				calcOptimalFlag = true

			case a := <- CompletedOrderChan:
				clearOrdersAtFloor(a, currHallOrdersChan, currCabOrdersChan, TurnOffLightsChan)
				calcOptimalFlag = true
		}

		if calcOptimalFlag {
			currOptimizationInputJson = encodeJson(currHallOrdersChan, currCabOrdersChan, currAllNodeStatesChan);
			outJson := runOptimizer(currOptimizationInputJson);
			json.Unmarshal(outJson, &optimalAssignedOrders);
			LocallyAssignedOrdersChan <- optimalAssignedOrders[string(localID)]
			calcOptimalFlag = false;
			// TODO why does it send two times?
		}
	}
}