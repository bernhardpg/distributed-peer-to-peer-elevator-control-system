package optimalAssigner

import (
	"os/exec"
	"fmt"
	"log"
	"os"
	"encoding/json"
	"../stateHandler"
)


type OptimalAssignerChannels struct {
	HallOrders chan [][] bool
	CabOrders chan [] bool
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

func encodeJson(currHallOrders [][]bool,
	// Encodes the data for HallRequstAssigner script, according to
	// Format required
	currCabOrders []bool, currAllElevStates map[stateHandler.NodeID] stateHandler.ElevState) ([]byte) {

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

// TODO export comment
func Assigner(numFloors int, HallOrdersChan <-chan [][] bool, CabOrdersChan <-chan [] bool,
	AllElevStatesChan <-chan map[stateHandler.NodeID] stateHandler.ElevState) {

	currHallOrders := make([][] bool, numFloors); 
	currCabOrders := make([] bool, numFloors);

	var currAllElevStates map[stateHandler.NodeID] stateHandler.ElevState;
	var currOptimizationInputJson []byte;
	var optimalAssignedOrders map[string]interface{};
	calcOptimalFlag := false;

	for {
		select {
			case a := <- HallOrdersChan:
				currHallOrders = a;
				calcOptimalFlag = true;
			case a := <- CabOrdersChan:
				currCabOrders = a;
				calcOptimalFlag = true;
			case a := <- AllElevStatesChan:
				currAllElevStates = a;
				calcOptimalFlag = true;
		}

		// TODO is this good style?
		if calcOptimalFlag {	
			currOptimizationInputJson = encodeJson(currHallOrders, currCabOrders, currAllElevStates);
			outJson := runOptimizer(currOptimizationInputJson);
			fmt.Println(string(outJson));
			json.Unmarshal(outJson, &optimalAssignedOrders);

			calcOptimalFlag = false;
		}
	}
}

func runOptimizer(currOptimizationInputJson []byte) ([]byte){
	// Get current working directory
	dir, err := os.Getwd();

	if err != nil {
		log.Fatal(err);
	}

	// Run external script with json data
	cmd := exec.Command("sh", "-c",
		dir + "/optimalAssigner/hall_request_assigner --input '" + string(currOptimizationInputJson) + "'");
	
	outJson, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	return outJson;
}