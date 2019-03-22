package optimalOrderAssigner

import (
	"../elevio"
	"../fsm"
	"../nodeStatesHandler"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"reflect"
)

type OptimalOrderAssignerChannels struct {
	HallOrdersChan            chan [][]bool
	CabOrdersChan             chan []bool
	NewOrderChan              chan elevio.ButtonEvent // TODO move to consensus module
	LocallyAssignedOrdersChan chan [][]bool
	CompletedOrderChan        chan int

}

type singleNodeStateJSON struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type optimizationInputJSON struct {
	HallRequests [][]bool                       `json:"hallRequests"`
	States       map[string]singleNodeStateJSON `json:"states"`
}

// Encodes the data for HallRequstAssigner script, according to
// format required by optimization script
func encodeJSON(
	currHallOrders [][]bool,
	currCabOrders []bool,
	currAllNodeStates map[nodeStatesHandler.NodeID] fsm.NodeState)([]byte) {

	// TODO change currCabOrders to allCabOrders

	currStates := make(map[string]singleNodeStateJSON)

	for currID, currNodeState := range currAllNodeStates {
		currBehaviour := "";
		currDirection := "";

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

	scriptName := "/optimalOrderAssigner/hall_request_assigner"
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

// Sets the order in the correct order matrix (if the order is not already set)
// and tell lightio to turn on the corresponding light (if the order is set).
// @return: true if order is set, false if order was already set
func setOrder(
	buttonPress elevio.ButtonEvent,
	hallOrders [][]bool,
	cabOrders []bool) bool {

	if buttonPress.Button == elevio.BT_Cab {
		if cabOrders[buttonPress.Floor] {
			return false
		}
		cabOrders[buttonPress.Floor] = true

	} else {
		if hallOrders[buttonPress.Floor][buttonPress.Button] {
			return false
		}
		hallOrders[buttonPress.Floor][buttonPress.Button] = true
	}

	// TODO remove commented light code
	// TurnOnLightsChan <- buttonPress
	return true
}

// Clear all orders in the correct order matrix and tell lightio to turn off the corresponding lights
func clearOrdersAtFloor(
	floor int,
	hallOrders [][]bool,
	cabOrders []bool) {

	cabOrders[floor] = false
	hallOrders[floor] = []bool{false, false}

	// TODO remove commented light code
	/*// Clear all button lights on floor
	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
		TurnOffLightsChan <- elevio.ButtonEvent{
			Floor:  floor,
			Button: orderType,
		}
	}*/
}

// Assigner ...
// Will calculate and assign confirmed orders to the current node each time new state data or
// new confirmed orders enters the system.
// The new calculated orders are sent to the fsm.
// The optimal distribution of orders are calculated using an external script, utilizing the state
// information on each node in addition to all the confirmed orders in the system.
func Assigner(
	localID nodeStatesHandler.NodeID,
	numFloors int,
	LocallyAssignedOrdersChan chan<- [][]bool,
	//NewOrderChan <-chan elevio.ButtonEvent, // Will be removed
	ConfirmedHallOrdersChan <-chan [][] bool,
	CompletedOrderChan <-chan int, // Will be removed
	AllNodeStatesChan <-chan map[nodeStatesHandler.NodeID]fsm.NodeState) {


	// Initialize variables
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

	optimize := false
	currAllNodeStates := make(map[nodeStatesHandler.NodeID] fsm.NodeState);
	var currOptimizationInputJSON []byte
	var optimalAssignedOrders map[string][][]bool

	// Order Assigner
	// (Handler for assigning all confirmed orders when new data enters the system)
	// --------

	for {
		select {

		// Optimize each time allNodeStates are updated
		case a := <- AllNodeStatesChan:

			// Don't react if no changes
			if reflect.DeepEqual(a, currAllNodeStates) {
				break
			}

			currAllNodeStates = a
			optimize = true

		/*case a := <- NewOrderChan:
			// Optimize if something is changed
			if setOrder(a, currHallOrders, currCabOrders) {
				optimize = true
			}*/

		// Optimize if the new order was not already in the system
		case a := <- ConfirmedHallOrdersChan:

			/*// TODO implement datatypes!
			if reflect.DeepEqual(a, currHallOrders) {
				break
			}*/

			// Why are these equal?? Because the underlying memory is the same

			currHallOrders = a
			optimize = true

		// Clear completed orders
		// (Note: Does not optimize because state is changed on completed orders)
		case a := <- CompletedOrderChan:
			clearOrdersAtFloor(a, currHallOrders, currCabOrders)

		default:
			// TODO is this necessary?
		}

		// Calculate optimal AssignedLocalOrders when new data has arrived and states have been initialized 
		if optimize && len(currAllNodeStates) != 0 {
			optimize = false

			// Calculate new optimalAssignedOrders time a message is received
			currOptimizationInputJSON = encodeJSON(currHallOrders, currCabOrders, currAllNodeStates);
			outJSON := runOptimizer(currOptimizationInputJSON);
			json.Unmarshal(outJSON, &optimalAssignedOrders);

			currLocallyAssignedOrders := optimalAssignedOrders[string(localID)]

			LocallyAssignedOrdersChan <- currLocallyAssignedOrders
		}
	}
}
