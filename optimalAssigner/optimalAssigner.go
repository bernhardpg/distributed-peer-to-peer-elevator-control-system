package optimalAssigner

import (
	"os/exec"
	"../fsm"
	"fmt"
	"log"
	"os"
	"time"
	"encoding/json"
)


type OptimalAssignerChns struct {
	HallOrdersChan chan [][] bool
	CabOrdersChan chan [] bool
	ElevStateChan chan fsm.ElevStateObject
}
	/*

	Goal json object input:
	{
	    "hallRequests" : 
	        [[Boolean, Boolean], ...],
	    "states" : 
	        {
	            "id_1" : {
	                "behaviour"     : < "idle" | "moving" | "doorOpen" >
	                "floor"         : NonNegativeInteger
	                "direction"     : < "up" | "down" | "stop" >
	                "cabRequests"   : [Boolean, ...]
	            },
	            "id_2" : {...}
	        }
	}

*/ 

	/* Json output 

	{
    "id_1" : [[Boolean, Boolean], ...],
    "id_2" : ...
	}

	*/

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
	currCabOrders []bool, currElevState fsm.ElevStateObject) ([]byte) {

	currStates := make(map[string] singleElevStateJson);

	// TODO these need to not be hardcoded!
	currBehaviour := "idle";
	currDirection := "up";

	//switch ElevStateObject.dir

	// TODO will need to iterate through all elements
	currStates["id_curr"] = singleElevStateJson {
		Behaviour: currBehaviour,
		Floor: currElevState.Floor,
		Direction: currDirection,
		CabRequests: currCabOrders,
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
	ElevStateChan <-chan fsm.ElevStateObject) {
	// TODO change package time
	encodePeriod := 500 * time.Millisecond;
	encodeTimer := time.NewTimer(encodePeriod);

	currHallOrders := make([][] bool, numFloors); 
	currCabOrders := make([] bool, numFloors);

	// TODO move to stateHandler!!
	currElevState := fsm.ElevStateObject {
		State: fsm.InitState,
		Floor: -1,
		Dir: fsm.Up,
	}

	var currOptDataJson []byte;
	var optimalAssignedOrders map[string]interface{};

	for {
		select {
			case a := <- HallOrdersChan:
				currHallOrders = a;
			case a := <- CabOrdersChan:
				currCabOrders = a;
			case a := <- ElevStateChan:
				currElevState = a;
			case <- encodeTimer.C:
				// TODO remove timer
				currOptDataJson = encodeJson(currHallOrders, currCabOrders, currElevState);
				outJson := runOptimizer(currOptDataJson);
				json.Unmarshal(outJson, &optimalAssignedOrders);

				// Optimally assigned orders!

				fmt.Println(optimalAssignedOrders);
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
	// TODO stop script from running in new window!
	cmd := exec.Command("sh", "-c",
		dir + "/optimalAssigner/hall_request_assigner --input '" + string(currOptDataJson) + "'");
	
	outJson, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	return outJson;
}