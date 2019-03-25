package fsm

import (
	"../datatypes"
	"../elevio"
	"time"
	"fmt"
)

// Channels ...
// Channels used for communication betweem the Elevator FSM and other modules
type Channels struct {
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
	Up OrderDir = iota
	Down   		
)

type NodeState struct {
	Behaviour NodeBehaviour
	Floor int
	Dir   OrderDir
}


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


func findFirstOrder(assignedOrders datatypes.AssignedOrdersMatrix) int {
	for floor := 0; floor < len(assignedOrders); floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			if assignedOrders[floor][orderType] {
				return floor
			}
		}
	}
	return -1
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

func ordersAhead(assignedOrders datatypes.AssignedOrdersMatrix, currFloor int, currDir OrderDir) bool {

	if currDir == Up {

		for floor := currFloor + 1; floor < len(assignedOrders); floor++ {
			fmt.Printf("(fsm) ordersAhead: iterating through floor %v from currFloor %v.\n", floor, currFloor)
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


func calculateDirection(currFloor int, requestedFloor int) OrderDir {
	if currFloor < requestedFloor {
		return Up
	}
		return Down
	}

func shouldStopAtFloor(currFloor int, numFloors int, currDir OrderDir, assignedOrders datatypes.AssignedOrdersMatrix) bool {

	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {

		if orderType == elevio.BT_HallUp && currDir == Down {
			continue
		} else if orderType == elevio.BT_HallDown && currDir == Up {
			continue
		}
		if assignedOrders[currFloor][orderType]{
			return true
		}
	}

	return !ordersAhead(assignedOrders, currFloor, currDir)		
}

func initiateMovement(currDir OrderDir){
	if currDir == Up {
		elevio.SetMotorDirection(elevio.MD_Up)
	} else {
		elevio.SetMotorDirection(elevio.MD_Down)
	}
}
func stopMovement(){
	elevio.SetMotorDirection(elevio.MD_Stop)
}
func openDoors(){
	elevio.SetDoorOpenLamp(true)
}
func closeDoors(){
	elevio.SetDoorOpenLamp(false)
}

// StateMachine ...
// GoRoutine for handling the states of a single elevator
func StateMachine(
	numFloors int,
	ArrivedAtFloorChan <-chan int,
	LocallyAssignedOrdersChan <-chan datatypes.AssignedOrdersMatrix,
	CompletedHallOrderChan chan<- int,
	CompletedCabOrderChan chan<- int,
	LocalNodeStateChan chan<- NodeState) {

	// Initialize variables	
	// -----
	doorOpenTime := 3 * time.Second
	currFloor := -1
	requestedFloor := -1
	currDir := Up
	doorTimer := time.NewTimer(0)

	var assignedOrders datatypes.AssignedOrdersMatrix

	// Initialize elevator
	// -----
	behaviour := InitState
	// Note: Elevator will be able to accept orders while initializing

	// Close doors and move elevator to first floor in direction Up 
	closeDoors()
	initiateMovement(currDir)
	
	
	// State selector
	// -----
	for {
		select {
		
		// Time to close doors
		case <- doorTimer.C:

			if behaviour == InitState {
				break
			}

			closeDoors()

			if !hasOrders(assignedOrders){
				behaviour = IdleState

			} else {

				//There are orders in the system, but but none ahead. Turn around!
				if !ordersAhead(assignedOrders, currFloor, currDir) {
					if currDir == Up {
						currDir = Down
					} else {
						currDir = Up
					}
				}


				initiateMovement(currDir)
				behaviour = MovingState

			}

			//State has changed, inform network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)


		// Receive optimally calculated orders for this node from optimalOrderAssigner 
		case a := <- LocallyAssignedOrdersChan:
			assignedOrders = a


		case a := <- ArrivedAtFloorChan:
			currFloor = a

			switch behaviour {

				// First floor in Up direction hit during init. Halt!
				case InitState:
					stopMovement()
					behaviour = IdleState
					

				case MovingState:

					// Excuses for stopping at floor: Cab orders, relevant hall orders, no orders ahead
					if shouldStopAtFloor(currFloor, numFloors, currDir, assignedOrders){

						stopMovement()
						openDoors()

						doorTimer.Reset(doorOpenTime)
						behaviour = DoorOpenState

						// Tell hallConsensus to wipe all orders at floor
						CompletedHallOrderChan <- currFloor
						CompletedCabOrderChan <- currFloor
						
					}
			}
			// Changes to floor and state have been made, inform network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

			
		}


		// No active orders? Wait till orders come
		if !hasOrders(assignedOrders){
			continue
		}

		

		switch behaviour {

		case IdleState:

			// A new order is present, findFirstOrder returns its floor  
			requestedFloor = findFirstOrder(assignedOrders)

			switch requestedFloor {

			// We're summoned to where we are. Open doors!
			case currFloor:
				
				openDoors()
				doorTimer.Reset(doorOpenTime)

				// Tell hallConsensus to wipe all orders at floor
				CompletedHallOrderChan <- currFloor
				CompletedCabOrderChan <- currFloor
				behaviour = DoorOpenState

			// Some other floor than our own is requested.
			default:

				currDir = calculateDirection(currFloor, requestedFloor)
				initiateMovement(currDir)
				behaviour = MovingState
			}

			// Changes to state have been made, inform network module
			transmitState(behaviour, currFloor, currDir, LocalNodeStateChan)

		
		case DoorOpenState:

			requestedFloor = findFirstOrder(assignedOrders)

			// Refresh door timer if summoned to where at with doors open
			if requestedFloor == currFloor {
				doorTimer.Reset(doorOpenTime)

				// Tell hallConsensus to wipe all orders at floor
				CompletedHallOrderChan <- currFloor
				CompletedCabOrderChan <- currFloor
			}

		}
	}
}
