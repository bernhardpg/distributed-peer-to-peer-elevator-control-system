package fsm

import (
	"../datatypes"
	"../elevio"
	"time"
	"fmt"
	"reflect"
)

// StateMachineChannels ...
// Channels used for communication with the Elevator FSM
type StateMachineChannels struct {
	ArrivedAtFloorChan chan int 
}

type NodeBehaviour int;
const (
	InitState NodeBehaviour = iota;
	IdleState
	DoorOpenState
	MovingState
)

type OrderDir int;
const (
	Up OrderDir = iota;
	Down;
)

type NodeState struct {
	Behaviour NodeBehaviour
	Floor int
	Dir   OrderDir
}


func calculateNextOrder(
	behaviour NodeBehaviour,
	currFloor int, currDir OrderDir,
	assignedOrders datatypes.AssignedOrdersMatrix) int {

	numFloors := len(assignedOrders);
	skipCurrFloor := 0;

	// Don't check currFloor if moving
	if behaviour == MovingState {
		skipCurrFloor = 1;
	}

	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == Up {
		// Look for orders in Up direction
		for floor := currFloor + skipCurrFloor; floor <= numFloors - 1; floor++ {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if orderType == elevio.BT_HallDown {
					// Skip orders of opposite directon
					continue;
				}
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
			}
		}
		// Check orders of opposite directon last
		for floor := numFloors - 1; floor >= currFloor + skipCurrFloor; floor-- {
			if assignedOrders[floor][elevio.BT_HallDown] == true {
				return floor;

			}
		}
	} else {
		// Look for orders in Down direction
		for floor := currFloor - skipCurrFloor; floor >= 0; floor-- {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if orderType == elevio.BT_HallUp {
					// Skip orders of opposite directon
					continue;
				}
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
			}
		}
		// Check orders of opposite directon last
		for floor := 0; floor <= currFloor - skipCurrFloor; floor++ {
			if assignedOrders[floor][elevio.BT_HallUp] == true {
				return floor;
			}
		}
	}

	// Check other directions if no orders are found
	if currDir == Up {
		return calculateNextOrder(behaviour, currFloor, Down, assignedOrders);
	}
	return calculateNextOrder(behaviour, currFloor, Up, assignedOrders);
}

func hasOrders(assignedOrders datatypes.AssignedOrdersMatrix) (bool) {
	for floor := 0; floor < len(assignedOrders); floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			if assignedOrders[floor][orderType] == true {
				return true;
			}
		}
	}

	return false;
}

func calculateDirection(
	numFloors int,
	currFloor int,
	currOrder int,
	currDir OrderDir) (OrderDir) {

	if currOrder == currFloor {
		// Change direction if elev is at top or bottom floor
		if currFloor == 0 {
			return Up;
		} else if currFloor == numFloors - 1 {
			return Down;
		}
		return currDir;
	}

	if currOrder > currFloor {
		return Up;
	}
	return Down
}

func transmitState(
	currState NodeBehaviour,
	currFloor int,
	currDir OrderDir,
	LocalNodeStateChan chan<- NodeState) {

	currNodeState := NodeState {
		Behaviour: currState,
		Floor: currFloor,
		Dir: currDir, 
	}

	LocalNodeStateChan <- currNodeState
}

// StateMachine ...
// GoRoutine for handling the states of a single elevator
func StateMachine(
	numFloors int,
	ArrivedAtFloorChan <-chan int,
	LocallyAssignedOrdersChan <-chan datatypes.AssignedOrdersMatrix,
	CompletedHallOrderChan chan<- int,
	CompletedCabOrderChan chan<- int,
	LocalNodeStateChan chan<- NodeState,
	CompletedOrderChan chan<- int) {

	// Initialize variables	
	// -----
	doorOpenTime := 3 * time.Second;
	currOrder := -1;
	currFloor := -1;
	currDir := Up;
	doorTimer := time.NewTimer(0);

	var assignedOrders datatypes.AssignedOrdersMatrix
	fmt.Println("(fsm) assignedOrders: ", assignedOrders)

	// Initialize elevator
	// -----
	behaviour := InitState
	nextBehaviour := behaviour
	// Note: Elevator will be able to accept orders while initializing


	// Used to transition to states when no channel action
	// or when transitioning from a state to the same state
	updateState := false;

	// Close doors and move elevator to first floor in direction Up 
	elevio.SetDoorOpenLamp(false)
	elevio.SetMotorDirection(elevio.MD_Up)
	
	// State selector
	// -----
	for {
		select {
		
		// Time to close door
		case <- doorTimer.C:


			// Don't react while initing
			if behaviour == InitState {
				break;
			}
	
			// Transition to next behaviour when door has been open
			// for desired period of time
			elevio.SetDoorOpenLamp(false)
			if hasOrders(assignedOrders) {
				nextBehaviour = MovingState
			} else {
				nextBehaviour = IdleState
			}

// NOTE! Will currently get stuck with open doors ebcause orders are not currently cleared

		// Receive optimally calculated orders for this node from optimalOrderAssigner 
		case a := <- LocallyAssignedOrdersChan:

			// Mark the current floor as complete if doors already open in desired floor
			// (Necessary to handle button spamming in same floor)
			if behaviour == DoorOpenState && currOrder == currFloor {
				CompletedHallOrderChan <- currFloor
//				CompletedCabOrderChan <- currFloor
				CompletedOrderChan <- currFloor // TODO remove
			}

			// Only react to changes
			if reflect.DeepEqual(a, assignedOrders) {
				break
			}

			// Transition to MovingState (where new currOrder and currDir will be calculated) if there are new orders
			assignedOrders = a
			if hasOrders(assignedOrders) && behaviour != DoorOpenState {
				nextBehaviour = MovingState
				updateState = true
			}

		// Elevator arrives at a floor
		case a := <- ArrivedAtFloorChan:
			currFloor = a;

			// Finish init sequence
			if behaviour == InitState {
				nextBehaviour = IdleState
			}

			// Open doors at desired floor and signal that the order is complete
			if currFloor == currOrder {
				CompletedHallOrderChan <- currFloor
//				CompletedCabOrderChan <- currFloor
				CompletedOrderChan <- currFloor // TODO remove
				nextBehaviour = DoorOpenState
				break;
			}

			// TransmitState everytime the elevator reaches a floor but doesn't stop 
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		default:
			// Required to make State Transition Handling work even when there are no channel action

		}

		// State Transition Handling
		// -----
		if nextBehaviour != behaviour || updateState {
			// Set new current behaviour
			switch nextBehaviour {

				// Transitioning to DoorOpenState will stop the elevator,
				// open the door (if not already opened) and restart the door timer
				case DoorOpenState:
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(doorOpenTime)
					updateState = false

				// Transitioning to IdleState stops the elevator
				case IdleState:
					elevio.SetMotorDirection(elevio.MD_Stop)

				// Transitioning to MovingState will always calculate new currOrder and currDir 
				case MovingState:
					currOrder = calculateNextOrder(behaviour, currFloor, currDir, assignedOrders)
					currDir = calculateDirection(numFloors, currFloor, currOrder, currDir)

					// Go directly to doorOpenState if already at desired floor and not moving
					if currOrder == currFloor && behaviour != MovingState {
						CompletedHallOrderChan <- currFloor
//						CompletedCabOrderChan <- currFloor
						CompletedOrderChan <- currFloor // TODO remove
						nextBehaviour = DoorOpenState
						updateState = true
						break
					}

					// Set motor direction
					if currDir == Up {
						elevio.SetMotorDirection(elevio.MD_Up)
					} else {
						elevio.SetMotorDirection(elevio.MD_Down)
					}
					
					// No need to change state immediately
					updateState = false
			}
			behaviour = nextBehaviour
			// Transmit behaviour each time behaviour is changed
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)
		}
	}
}

