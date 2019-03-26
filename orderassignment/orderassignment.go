package orderassignment

import (
	"../datatypes"
	"../elevio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
)

// Channels ...
// Used for communication between this module and other modules
type Channels struct {
	LocallyAssignedOrdersChan chan datatypes.AssignedOrdersMatrix
	PeerlistUpdateChan        chan []datatypes.NodeID
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
	currAllCabOrders datatypes.ConfirmedCabOrdersMap,
	currAllNodeStates map[datatypes.NodeID]datatypes.NodeState,
	peerlist []datatypes.NodeID) []byte {

	currStates := make(map[string]singleNodeStateJSON)

	for _, currID := range peerlist {
		currBehaviour := ""
		currDirection := ""

		// Initialize cabOrders to false if not yet defined
		// (The order distribution will quickly converge towards
		// the correct distribution, so this is not a problem)
		currNodeState := currAllNodeStates[currID]
		currCabOrders := currAllCabOrders[currID]
		if currCabOrders == nil {
			currCabOrders = make(datatypes.ConfirmedCabOrdersList, elevio.NumFloors)
		}

		switch currNodeState.Behaviour {

		case datatypes.IdleState, datatypes.InitState:
			currBehaviour = "idle"
			currDirection = "stop"

		case datatypes.MovingState:
			currBehaviour = "moving"

			switch currNodeState.Dir {
			case datatypes.Up:
				currDirection = "up"
			case datatypes.Down:
				currDirection = "down"
			}

		case datatypes.DoorOpenState:
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

// runOptimizer ...
// Runs the optimization script with the given JSON data
// @return: JSON object with optimal distribution of orders between
// all nodes in the system.
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
	PeerlistUpdateChan <-chan []datatypes.NodeID,
	LocallyAssignedOrdersChan chan<- datatypes.AssignedOrdersMatrix,
	ConfirmedHallOrdersChan <-chan datatypes.ConfirmedHallOrdersMatrix,
	ConfirmedCabOrdersChan <-chan datatypes.ConfirmedCabOrdersMap,
	AllNodeStatesChan <-chan map[datatypes.NodeID]datatypes.NodeState) {

	// Initialize variables
	//-------

	var currHallOrders datatypes.ConfirmedHallOrdersMatrix
	var currAllCabOrders datatypes.ConfirmedCabOrdersMap
	var peerlist []datatypes.NodeID

	optimize := false
	currAllNodeStates := make(map[datatypes.NodeID]datatypes.NodeState)
	var currOptimizationInputJSON []byte
	var optimalAssignedOrders map[string]datatypes.AssignedOrdersMatrix

	fmt.Println("(optimalassigner) Initialized")

	// Order Assigner
	// (Handler for assigning all confirmed orders when new data enters the system)
	// --------

	for {
		select {

		case a := <-PeerlistUpdateChan:
			peerlist = a
			optimize = true

		// Optimize each time allNodeStates are updated
		case a := <-AllNodeStatesChan:

			// Don't react if no changes
			if reflect.DeepEqual(a, currAllNodeStates) {
				break
			}

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

		// Receive new confirmedOrders from hallConsensus
		// Optimize if the new order is not already in the system
		case a := <-ConfirmedCabOrdersChan:
			// Avoid double calculation when hit desired floor
			if reflect.DeepEqual(a, currAllCabOrders) {
				break
			}

			currAllCabOrders = a
			optimize = true

		default:
		}

		// Calculate optimal assigned order for the local node when
		// new data has arrived and the states have been initialized
		if optimize && len(currAllNodeStates) != 0 {
			optimize = false

			// Encode information as JSON, pass it to the optimizer script,
			// and finally extract the optimal orders for the current node

			currOptimizationInputJSON = encodeJSON(currHallOrders,
				currAllCabOrders, currAllNodeStates, peerlist)
			outJSON := runOptimizer(currOptimizationInputJSON)
			json.Unmarshal(outJSON, &optimalAssignedOrders)
			currLocallyAssignedOrders := optimalAssignedOrders[string(localID)]

			// Update the FSM with the new assigned orders
			LocallyAssignedOrdersChan <- currLocallyAssignedOrders
		}
	}
}
