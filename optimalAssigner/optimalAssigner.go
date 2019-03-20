package optimalAssigner

import (
	"os/exec"
	"log"
	"os"
	"encoding/json"
	"../stateHandler"
	"../elevio"
	"fmt"
	//"reflect"
)

type OptimalAssignerChannels struct {
	HallOrders chan [][] bool
	CabOrders chan [] bool
	NewOrder chan elevio.ButtonEvent // TODO move to consensus module
	LocallyAssignedOrders chan [][] bool
	CompletedOrder chan int
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
func encodeJson(currHallOrders [][]bool,
	currCabOrders []bool, currAllElevStates map[stateHandler.NodeID] stateHandler.ElevState)([]byte) {

	currStates := make(map[string] singleElevStateJson);

	for currID, currElevState := range currAllElevStates {
		// TODO these need to not be hardcoded!
		currBehaviour := "idle";
		currDirection := "up";

		//switch ElevState.dir

		currStates[string(currID)] = singleElevStateJson {
			Behaviour: currBehaviour,
			Floor: currElevState.Floor,
			Direction: currDirection,
			CabRequests: currCabOrders,
		}
	}

	currOptimizationInput := optimizationInputJson {
		HallRequests: currHallOrders,
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
	fmt.Println("Setting order in local order matrix!")
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
	HallOrdersChan <-chan [][] bool,
	CabOrdersChan <-chan [] bool,
	LocallyAssignedOrdersChan chan<- [][]bool,
	NewOrderChan <-chan elevio.ButtonEvent,
	CompletedOrderChan <-chan int,
	AllElevStatesChan <-chan map[stateHandler.NodeID] stateHandler.ElevState) { 


	// Initialize empty matrices
	//-------
	currHallOrders := make([][] bool, numFloors); 
	currCabOrders := make([] bool, numFloors);

	for floor := range currHallOrders {
		currHallOrders[floor] = make([] bool, 2)
	}

	for floor := range currHallOrders {
		for orderType := range currHallOrders[floor] {
			currHallOrders[floor][orderType] = false
		}
		currCabOrders[floor] = false
	}

	currAllElevStates := make(map[stateHandler.NodeID] stateHandler.ElevState);
	var currOptimizationInputJson []byte;
	var optimalAssignedOrders map[string] [][]bool;
	calcOptimalFlag := false;

	for {
		select {
			/*case a := <- HallOrdersChan:
				currHallOrders = a;
				calcOptimalFlag = true;
			case a := <- CabOrdersChan:
				currCabOrders = a;
				calcOptimalFlag = true;*/

			case a := <- AllElevStatesChan:
				fmt.Println("OptimalAssigner: received allElevStates!")
				currAllElevStates = a
				calcOptimalFlag = true
				//fmt.Println("Updating state!:")
				//fmt.Println(currAllElevStates)
				//fmt.Println(a)

			case a := <- NewOrderChan:
				setOrder(a, currHallOrders, currCabOrders)
				calcOptimalFlag = true

			case a := <- CompletedOrderChan:
				clearOrdersAtFloor(a, currHallOrders, currCabOrders)
				fmt.Println("optimalAssigner: Cleared orders at floor")
				calcOptimalFlag = true
		}

		if calcOptimalFlag {
			currOptimizationInputJson = encodeJson(currHallOrders, currCabOrders, currAllElevStates);
			outJson := runOptimizer(currOptimizationInputJson);
			json.Unmarshal(outJson, &optimalAssignedOrders);
			//fmt.Println(optimalAssignedOrders[string(localID)]);

			fmt.Println("OptimalAssigner: Sending new locally assigned orders to fsm")
			LocallyAssignedOrdersChan <- optimalAssignedOrders[string(localID)]
			fmt.Println("OptimalAssigner: Sent new locally assigned orders")
			calcOptimalFlag = false;

			// TODO why does it send two times?
		}
	}
}