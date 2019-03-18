package optimalAssigner

import (
	"os/exec"
	"fmt"
	"log"
	"os"
	"time"
	"encoding/json"
	"../stateHandler"
)


type OptimalAssignerChannels struct {
	HallOrders chan [][] bool
	CabOrders chan [] bool
	ElevState chan stateHandler.ElevState
}

type singleElevStateJson struct {
	Behaviour string 	`json:"behaviour"`
	Floor int 			`json:"floor"`
	Direction string 	`json:"direction"`
	CabRequests []bool 	`json:"cabRequests"`
}

type optimizationDataJson struct {
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

	currOptData := optimizationDataJson {
		HallRequests: currHallOrders,
		States: currStates,
	}

	currOptDataJson,_ := json.Marshal(currOptData);
	return currOptDataJson;
}

// TODO export comment
func Assigner(numFloors int, HallOrdersChan <-chan [][] bool, CabOrdersChan <-chan [] bool,
	ElevStateChan <-chan stateHandler.ElevState, AllElevStatesChan <-chan map[stateHandler.NodeID] stateHandler.ElevState) {
	// TODO change package time
	encodePeriod := 500 * time.Millisecond;
	encodeTimer := time.NewTimer(encodePeriod);


	currHallOrders := make([][] bool, numFloors); 
	currCabOrders := make([] bool, numFloors);

	var currAllElevStates map[stateHandler.NodeID] stateHandler.ElevState;

	var currOptDataJson []byte;
	var optimalAssignedOrders map[string]interface{};

	for {
		select {
			case a := <- HallOrdersChan:
				currHallOrders = a;
			case a := <- CabOrdersChan:
				currCabOrders = a;
			case a := <- ElevStateChan:
				fmt.Println(a);
				//TODO REMOVE THIS
			case a := <- AllElevStatesChan:
				currAllElevStates = a;
			case <- encodeTimer.C:
				// TODO remove timer
				currOptDataJson = encodeJson(currHallOrders, currCabOrders, currAllElevStates);
				outJson := runOptimizer(currOptDataJson);
				fmt.Println(string(outJson));
				json.Unmarshal(outJson, &optimalAssignedOrders);

				// Optimally assigned orders!

				//fmt.Println(optimalAssignedOrders);
				encodeTimer.Reset(encodePeriod);
		}
	}
}

func runOptimizer(currOptDataJson []byte) ([]byte){
	// Get current working directory
	dir, err := os.Getwd();

	if err != nil {
		log.Fatal(err);
	}

	// Run external script
	cmd := exec.Command("sh", "-c",
		dir + "/optimalAssigner/hall_request_assigner --input '" + string(currOptDataJson) + "'");
	
	outJson, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	return outJson;
}