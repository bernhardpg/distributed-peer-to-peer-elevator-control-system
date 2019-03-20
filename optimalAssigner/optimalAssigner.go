package optimalAssigner

import (
	"os/exec"
	"log"
	"os"
	"encoding/json"
	"../stateHandler"
	"../elevio"
)

type OptimalAssignerChannels struct {
	HallOrdersChan chan [][] bool
	CabOrdersChan chan [] bool
	NewOrderChan chan elevio.ButtonEvent // TODO move to consensus module
	LocallyAssignedOrdersChan chan [][] bool
	CompletedOrderChan chan int
}

type singleElevStateJson struct {
	Behaviour string 	`json:"behaviour"`
	Floor int 			`json:"floor"`
	Direction string 	`json:"direction"`
	CabRequests []bool 	`json:"cabRequests"`
}

type optimizationInputJson struct {
	// Following format req to use HallRequestAssigner (written in D)
	HallRequests [][]bool 					`json:"hallRequests"`
	States map[string] singleElevStateJson 	`json:"states"`
}

// Encodes the data for HallRequstAssigner script, according to
// Format required
func encodeJson(currHallOrdersChan [][]bool,
	currCabOrdersChan []bool, currAllElevStatesChan map[stateHandler.NodeID] stateHandler.ElevState)([]byte) {

	currStates := make(map[string] singleElevStateJson);

	for currID, currElevState := range currAllElevStatesChan {
		// TODO these need to not be hardcoded!
		currBehaviour := "idle";
		currDirection := "up";

		//switch ElevState.dir

		currStates[string(currID)] = singleElevStateJson {
			Behaviour: currBehaviour,
			Floor: currElevState.Floor,
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

func setOrder(buttonPress elevio.ButtonEvent, hallOrders [][]bool, cabOrders []bool) {
	if buttonPress.Button == elevio.BT_Cab {
		cabOrders[buttonPress.Floor] = true
	} else {
		hallOrders[buttonPress.Floor][buttonPress.Button] = true;
	}

	// TODO set lights
	// TODO send orders to fsm!
}

func clearOrdersAtFloor(floor int, hallOrders [][]bool, cabOrders []bool) {
	cabOrders[floor] = false
	hallOrders[floor] = []bool{false, false}
}

func Assigner(localID stateHandler.NodeID, numFloors int,
	HallOrdersChanChan <-chan [][] bool,
	CabOrdersChanChan <-chan [] bool,
	LocallyAssignedOrdersChanChan chan<- [][]bool,
	NewOrderChanChan <-chan elevio.ButtonEvent,
	CompletedOrderChanChan <-chan int,
	AllElevStatesChanChan <-chan map[stateHandler.NodeID] stateHandler.ElevState) { 


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

	currAllElevStatesChan := make(map[stateHandler.NodeID] stateHandler.ElevState);
	var currOptimizationInputJson []byte;
	var optimalAssignedOrders map[string] [][]bool;
	calcOptimalFlag := false;

	for {
		select {
			case a := <- AllElevStatesChanChan:
				currAllElevStatesChan = a
				calcOptimalFlag = true

			case a := <- NewOrderChanChan:
				setOrder(a, currHallOrdersChan, currCabOrdersChan)
				calcOptimalFlag = true

			case a := <- CompletedOrderChanChan:
				clearOrdersAtFloor(a, currHallOrdersChan, currCabOrdersChan)
				calcOptimalFlag = true
		}

		if calcOptimalFlag {
			currOptimizationInputJson = encodeJson(currHallOrdersChan, currCabOrdersChan, currAllElevStatesChan);
			outJson := runOptimizer(currOptimizationInputJson);
			json.Unmarshal(outJson, &optimalAssignedOrders);
			LocallyAssignedOrdersChanChan <- optimalAssignedOrders[string(localID)]
			calcOptimalFlag = false;
			// TODO why does it send two times?
		}
	}
}