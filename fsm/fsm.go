package fsm

import (
	"../datatypes"
	"../elevio"
	"fmt"
	"time"
)

// Channels ...
// Channels used for communication betweem the Elevator FSM and other modules
type Channels struct {
	ArrivedAtFloorChan          chan int
	ToggleNetworkVisibilityChan chan bool
}

// NodeBehaviour ...
// Contains the current behaviour of the node.
type NodeBehaviour int

// Possible node behaviours
const (
	// InitState ...
	// Used for initializing
	// (Either after a restart or after being physically obstructed)
	InitState NodeBehaviour = iota

	// IdleState ...
	// Node is standing still without orders.
	IdleState

	// DoorOpenState ...
	// Node is standing in a floor with the doors open.
	DoorOpenState

	// MovingState ...
	// Node is moving.
	MovingState
)

// OrderDir ...
// Which direction the node is currently moving.
// (Will also decide which direction the node will look for
// new orders first).
type OrderDir int

const (
	Up OrderDir = iota
	Down
)

// NodeState ...
// Contains all the state information of a node
type NodeState struct {
	Behaviour NodeBehaviour
	Floor     int
	Dir       OrderDir
}

// hasOrders ...
// @return: true if the node has any orders, false otherwise
func hasOrders(assignedOrders datatypes.AssignedOrdersMatrix) bool {
	for floor := 0; floor < len(assignedOrders); floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			if assignedOrders[floor][orderType] {
				return true
			}
		}
	}
	return false
}

// hasOrdersAtFloor ...
// @return: true if the node has an order at the given floor, false otherwise
func hasOrderAtFloor(
	assignedOrders datatypes.AssignedOrdersMatrix,
	floor int) bool {
	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
		if assignedOrders[floor][orderType] {
			return true
		}
	}
	return false
}

// transmitState ...
// Transmits the current local node state to the nodestates handler
func transmitState(
	currState NodeBehaviour,
	currFloor int,
	currDir OrderDir,
	LocalNodeStateChan chan<- NodeState) {

	currNodeState := NodeState{
		Behaviour: currState,
		Floor:     currFloor,
		Dir:       currDir,
	}
	LocalNodeStateChan <- currNodeState
}

// ordersAhead ...
// @return: true if there are any orders in the given direction, false otherwise
func ordersAhead(
	assignedOrders datatypes.AssignedOrdersMatrix,
	currFloor int,
	currDir OrderDir) bool {

	if currDir == Up {

		for floor := currFloor + 1; floor < len(assignedOrders); floor++ {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if assignedOrders[floor][orderType] {
					return true
				}
			}
		}

	} else {

		for floor := currFloor - 1; floor >= 0; floor-- {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if assignedOrders[floor][orderType] {
					return true
				}
			}
		}
	}

	return false
}

// calculateDirection ...
// @return: Up or Down depending on whether there are orders below or above
// the current floor, considering which direction the node is currently
// searching for orders
func calculateDirection(
	assignedOrders datatypes.AssignedOrdersMatrix,
	currFloor int, currDir OrderDir) OrderDir {

	if !ordersAhead(assignedOrders, currFloor, currDir) {
		if currDir == Up {
			return Down
		}
		return Up
	}
	return currDir
}

// shouldStopAtFloor ...
// @return: true if there is a cab order or a hall order (in the same direction a
// the elevator is currently moving) in the given floor, or if there
// are no orders ahead in the direction the elevator is moving.
func shouldStopAtFloor(
	currFloor int,
	numFloors int,
	currDir OrderDir,
	assignedOrders datatypes.AssignedOrdersMatrix) bool {

	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {

		if orderType == elevio.BT_HallUp && currDir == Down {
			continue
		} else if orderType == elevio.BT_HallDown && currDir == Up {
			continue
		}
		if assignedOrders[currFloor][orderType] {
			return true
		}
	}

	return !ordersAhead(assignedOrders, currFloor, currDir)
}

// Wrapper functions for controlling the elevator hardware
// -----
func initiateMovement(currDir OrderDir) {
	if currDir == Up {
		elevio.SetMotorDirection(elevio.MD_Up)
	} else {
		elevio.SetMotorDirection(elevio.MD_Down)
	}
}
func stopMovement() {
	elevio.SetMotorDirection(elevio.MD_Stop)
}
func openDoors() {
	elevio.SetDoorOpenLamp(true)
}
func closeDoors() {
	elevio.SetDoorOpenLamp(false)
}

// StateMachine ...
// GoRoutine acting as the Finite State Machine of a single node
func StateMachine(
	numFloors int,
	ArrivedAtFloorChan <-chan int,
	ToggleNetworkVisibilityChan chan<- bool,
	LocallyAssignedOrdersChan <-chan datatypes.AssignedOrdersMatrix,
	CompletedHallOrderChan chan<- int,
	CompletedCabOrderChan chan<- int,
	LocalNodeStateChan chan<- NodeState) {

	// Initialize variables
	// -----
	doorOpenTime := 3 * time.Second
	timeoutTime := 4 * time.Second

	currFloor := -1
	currDir := Up

	doorTimer := time.NewTimer(0)

	// Start obstruction timer on init
	obstructionTimer := time.NewTimer(timeoutTime)

	// Go offline until initialized
	ToggleNetworkVisibilityChan <- false

	var assignedOrders datatypes.AssignedOrdersMatrix

	// Initialize elevator
	// (Close doors and move to first floor in Up direction)
	// -----
	behaviour := InitState
	closeDoors()
	initiateMovement(currDir)

	fmt.Println("(fsm) Initialized")

	// Finite State Machine
	// -----
	for {
		select {

		// Possible obstruction, the elevator should have hit a floor by now
		case <-obstructionTimer.C:
			if behaviour != MovingState && behaviour != InitState {
				break
			}

			behaviour = InitState
			initiateMovement(currDir)
			obstructionTimer.Reset(timeoutTime)

			// Don't show on network when obstructed
			// (Will make the other nodes redistribute
			// the orders of this node)
			ToggleNetworkVisibilityChan <- false

		// Time to close doors and transition to another state
		case <-doorTimer.C:
			if behaviour == InitState {
				break
			}

			closeDoors()

			// Move to IdleState if there are no orders,
			// change to MovingState if there are.
			if !hasOrders(assignedOrders) {
				behaviour = IdleState
			} else {
				currDir = calculateDirection(assignedOrders, currFloor, currDir)
				initiateMovement(currDir)
				behaviour = MovingState

				// Start obstruction timer every time the node
				// transitions to MovingState
				obstructionTimer.Reset(timeoutTime)

			}

			// The node state has changed, inform the network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		// Receive (optimally) assigned orders for this node from the
		// optimal order assigner
		case a := <-LocallyAssignedOrdersChan:
			assignedOrders = a

		// Transition to correct state when arriving in new floor.
		case a := <-ArrivedAtFloorChan:
			currFloor = a

			// Reset the obstruction timer when the node arrives at a floor
			obstructionTimer.Reset(timeoutTime)

			switch behaviour {

			// Stop at first defined floor and go online when initialized
			case InitState:
				stopMovement()
				behaviour = IdleState
				ToggleNetworkVisibilityChan <- true

			// Transition from MovingState to DoorOpenState if the node
			// should stop at this floor
			case MovingState:
				if shouldStopAtFloor(currFloor, numFloors, currDir, assignedOrders) {
					stopMovement()
					openDoors()
					doorTimer.Reset(doorOpenTime)
					behaviour = DoorOpenState

					// Tell hallConsensus to wipe all orders at floor
					CompletedHallOrderChan <- currFloor
					CompletedCabOrderChan <- currFloor
				}
			}
			// The node state has changed, inform the network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		}

		// A new message has arrived on the channels, handle the state
		// transitioning if there are any orders in the system.
		// -----

		if !hasOrders(assignedOrders) {
			continue
		}

		switch behaviour {
		case IdleState:

			// The node is summoned to where it is, open doors!
			if hasOrderAtFloor(assignedOrders, currFloor) {
				openDoors()
				doorTimer.Reset(doorOpenTime)

				// Tell hallConsensus to wipe all orders at floor
				CompletedHallOrderChan <- currFloor
				CompletedCabOrderChan <- currFloor
				behaviour = DoorOpenState

			} else {
				// There are orders present, but not at the current floor.
				// Change dir if they're not ahead of the node.
				currDir = calculateDirection(assignedOrders, currFloor, currDir)

				initiateMovement(currDir)

				behaviour = MovingState
				// Start obstruction timer everytime the node starts moving
				obstructionTimer.Reset(timeoutTime)
			}

			// The node state has changed, inform the network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		case DoorOpenState:

			// Refresh door timer if summoned to the current floor, and
			// doors are already open
			if hasOrderAtFloor(assignedOrders, currFloor) {
				doorTimer.Reset(doorOpenTime)

				// Tell hallConsensus to wipe all orders at floor
				CompletedHallOrderChan <- currFloor
				CompletedCabOrderChan <- currFloor
			}

		}
	}
}
