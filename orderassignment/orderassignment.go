package orderassignment

import (
	"../datatypes"
	"../fsm"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"reflect"
)

// Channels ...
// Used for communication between this module and other modules
type Channels struct {
	LocallyAssignedOrdersChan chan datatypes.AssignedOrdersMatrix
}

type singleNodeStateJSON struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type optimizationInputJSON struct {
	HallRequests datatypes.ConfirmedHallOrdersMatrix `json:"hallRequests"`
	States       map[string]singleNodeStateJSON      `json:"states"`
}

// Encodes the data for HallRequstAssigner script, according to
// format required by optimization script
func encodeJSON(
	currHallOrders datatypes.ConfirmedHallOrdersMatrix,
	currCabOrders []bool,
	currAllNodeStates map[datatypes.NodeID]fsm.NodeState) []byte {

	// TODO change currCabOrders to allCabOrders

	currStates := make(map[string]singleNodeStateJSON)

	for currID, currNodeState := range currAllNodeStates {
		currBehaviour := ""
		currDirection := ""

		switch currNodeState.Behaviour {

		case fsm.IdleState, fsm.InitState:
			currBehaviour = "idle"
			currDirection = "stop"

		case fsm.MovingState:
			currBehaviour = "moving"

			switch currNodeState.Dir {
			case fsm.Up:
				currDirection = "up"
			case fsm.Down:
				currDirection = "down"
			}

		case fsm.DoorOpenState:
			currBehaviour = "doorOpen"
			currDirection = "stop"
		}

		currStates[string(currID)] = singleNodeStateJSON{
			Behaviour:   currBehaviour,
			Floor:       currNodeState.Floor,
			Direction:   currDirection,
			CabRequests: currCabOrders,
		}
	}

	currOptimizationInput := optimizationInputJSON{
		HallRequests: currHallOrders,
		States:       currStates,
	}


	currOptimizationInputJSON, _ := json.Marshal(currOptimizationInput)

	return currOptimizationInputJSON
}

// Runs the optimization script with the given JSON data
// @return: JSON object with optimal distribution of orders between nodes
func runOptimizer(currOptimizationInputJSON []byte) []byte {

	// Get current working directory
	dir, err := os.Getwd()

	if err != nil {
		log.Fatal(err)
	}

	scriptName := "/orderassignment/hall_request_assigner"
	params := "--includeCab --clearRequestType all"
	input := " --input '" + string(currOptimizationInputJSON) + "'"

	// Run external script with JSON data
	cmd := exec.Command("sh", "-c",
		dir+scriptName+" "+params+" "+input)

	outJSON, err := cmd.Output()

	if err != nil {
		log.Fatal(err)
	}

	return outJSON
}

// OptimalAssigner ...
// Will calculate and assign confirmed orders to the current node each time new state data or
// new confirmed orders enters the system.
// The new calculated orders are sent to the fsm.
// The optimal distribution of orders are calculated using an external script, utilizing the state
// information on each node in addition to all the confirmed orders in the system.
func OptimalAssigner(
	localID datatypes.NodeID,
	numFloors int,
	LocallyAssignedOrdersChan chan<- datatypes.AssignedOrdersMatrix,
	ConfirmedHallOrdersChan <-chan datatypes.ConfirmedHallOrdersMatrix,
	AllNodeStatesChan <-chan map[datatypes.NodeID]fsm.NodeState) {

	// Initialize variables
	//-------

	var currHallOrders datatypes.ConfirmedHallOrdersMatrix
	currCabOrders := make([]bool, numFloors)

	for floor := range currHallOrders {
		for orderType := range currHallOrders[floor] {
			currHallOrders[floor][orderType] = false
		}
		currCabOrders[floor] = false
	}

	optimize := false
	currAllNodeStates := make(map[datatypes.NodeID]fsm.NodeState)
	var currOptimizationInputJSON []byte
	var optimalAssignedOrders map[string]datatypes.AssignedOrdersMatrix

	// Order Assigner
	// (Handler for assigning all confirmed orders when new data enters the system)
	// --------

	for {
		select {

		// Optimize each time allNodeStates are updated
		case a := <-AllNodeStatesChan:

			// TODO fix datatypes
			/*// Don't react if no changes
			if reflect.DeepEqual(a, currAllNodeStates) {
				break
			}*/

			currAllNodeStates = a
			optimize = true

		// Receive new confirmedOrders from hallConsensus
		// Optimize if the new order is not already in the system
		case a := <-ConfirmedHallOrdersChan:

			// Avoid double calculation when hit desired floor
			if reflect.DeepEqual(a, currHallOrders) {
				break
			}

			currHallOrders = a
			optimize = true

		default:
			// TODO is this necessary?
		}

		// Calculate optimal AssignedLocalOrders when new data has arrived and states have been initialized
		if optimize && len(currAllNodeStates) != 0 {
			optimize = false

			// Calculate new optimalAssignedOrders time a message is received
			currOptimizationInputJSON = encodeJSON(currHallOrders, currCabOrders, currAllNodeStates)
			outJSON := runOptimizer(currOptimizationInputJSON)
			json.Unmarshal(outJSON, &optimalAssignedOrders)

			currLocallyAssignedOrders := optimalAssignedOrders[string(localID)]

			LocallyAssignedOrdersChan <- currLocallyAssignedOrders
		}
	}
}
