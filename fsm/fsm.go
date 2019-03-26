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
	currState datatypes.NodeBehaviour,
	currFloor int,
	currDir datatypes.NodeDir,
	LocalNodeStateChan chan<- datatypes.NodeState) {

	currNodeState := datatypes.NodeState{
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
	currDir datatypes.NodeDir) bool {

	if currDir == datatypes.Up {

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
// @return: datatypes.Up or datatypes.Down depending on whether there are orders below or above
// the current floor, considering which direction the node is currently
// searching for orders
func calculateDirection(
	assignedOrders datatypes.AssignedOrdersMatrix,
	currFloor int, currDir datatypes.NodeDir) datatypes.NodeDir {

	if !ordersAhead(assignedOrders, currFloor, currDir) {
		if currDir == datatypes.Up {
			return datatypes.Down
		}
		return datatypes.Up
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
	currDir datatypes.NodeDir,
	assignedOrders datatypes.AssignedOrdersMatrix) bool {

	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {

		if orderType == elevio.BT_HallUp && currDir == datatypes.Down {
			continue
		} else if orderType == elevio.BT_HallDown && currDir == datatypes.Up {
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
func initiateMovement(currDir datatypes.NodeDir) {
	if currDir == datatypes.Up {
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
	LocalNodeStateChan chan<- datatypes.NodeState) {

	// Initialize variables
	// -----
	doorOpenTime := 3 * time.Second
	timeoutTime := 4 * time.Second

	currFloor := -1
	currDir := datatypes.Up

	doorTimer := time.NewTimer(0)

	// Start obstruction timer on init
	obstructionTimer := time.NewTimer(timeoutTime)

	// Go offline until initialized
	ToggleNetworkVisibilityChan <- false

	var assignedOrders datatypes.AssignedOrdersMatrix

	// Initialize elevator
	// (Close doors and move to first floor in datatypes.Up direction)
	// -----
	behaviour := datatypes.InitState
	closeDoors()
	initiateMovement(currDir)

	fmt.Println("(fsm) Initialized")

	// Finite State Machine
	// -----
	for {
		select {

		// Possible obstruction, the elevator should have hit a floor by now
		case <-obstructionTimer.C:
			if behaviour != datatypes.MovingState && behaviour != datatypes.InitState {
				break
			}

			behaviour = datatypes.InitState
			initiateMovement(currDir)
			obstructionTimer.Reset(timeoutTime)

			// Don't show on network when obstructed
			// (Will make the other nodes redistribute
			// the orders of this node)
			ToggleNetworkVisibilityChan <- false

		// Time to close doors and transition to another state
		case <-doorTimer.C:
			if behaviour == datatypes.InitState {
				break
			}

			closeDoors()

			// Move to datatypes.IdleState if there are no orders,
			// change to datatypes.MovingState if there are.
			if !hasOrders(assignedOrders) {
				behaviour = datatypes.IdleState
			} else {
				currDir = calculateDirection(assignedOrders, currFloor, currDir)
				initiateMovement(currDir)
				behaviour = datatypes.MovingState

				// Start obstruction timer every time the node
				// transitions to datatypes.MovingState
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
			case datatypes.InitState:
				stopMovement()
				behaviour = datatypes.IdleState
				ToggleNetworkVisibilityChan <- true

			// Transition from datatypes.MovingState to datatypes.DoorOpenState if the node
			// should stop at this floor
			case datatypes.MovingState:
				if shouldStopAtFloor(currFloor, numFloors, currDir, assignedOrders) {
					stopMovement()
					openDoors()
					doorTimer.Reset(doorOpenTime)
					behaviour = datatypes.DoorOpenState

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
		case datatypes.IdleState:

			// The node is summoned to where it is, open doors!
			if hasOrderAtFloor(assignedOrders, currFloor) {
				openDoors()
				doorTimer.Reset(doorOpenTime)

				// Tell hallConsensus to wipe all orders at floor
				CompletedHallOrderChan <- currFloor
				CompletedCabOrderChan <- currFloor
				behaviour = datatypes.DoorOpenState

			} else {
				// There are orders present, but not at the current floor.
				// Change dir if they're not ahead of the node.
				currDir = calculateDirection(assignedOrders, currFloor, currDir)

				initiateMovement(currDir)

				behaviour = datatypes.MovingState
				// Start obstruction timer everytime the node starts moving
				obstructionTimer.Reset(timeoutTime)
			}

			// The node state has changed, inform the network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		case datatypes.DoorOpenState:

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
