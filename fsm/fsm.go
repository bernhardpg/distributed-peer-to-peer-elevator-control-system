package fsm

import (
	"../elevio"
	"time"
	"fmt"
	"reflect"
)

// StateMachineChannels ...
// Channels used for communication with the Elevator FSM
type StateMachineChannels struct {
	NewOrderChan chan elevio.ButtonEvent
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
	ID NodeID
	Behaviour NodeBehaviour
	Floor int
	Dir   OrderDir
}

type NodeID int;

/*// TODO move these data types to correct file!
type ReqState int;
const (
	Unknown ReqState = iota;
	Inactive
	PendingAck
	Confirmed
)

type nodeId int;

type Req struct {
	state ReqState;
	ackBy []nodeId;
}*/

	
// Initialize locally assigned orders matrix
/*var locallyAssignedOrders = make([][]Req, numFloors);
for i := range assignedOrders {
	assignedOrders[i] = make([]Req, numStates);
}

// Initialize all to unknown
for floor := range assignedOrders {
	for orderType := range assignedOrders[floor] {
		assignedOrders[floor][orderType].state = Unknown;
	}
}*/

func calculateNextOrder(currFloor int, currDir OrderDir, assignedOrders [][] bool) (int) {
	numFloors := len(assignedOrders);
	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == Up {
		for floor := currFloor; floor <= numFloors - 1; floor++ {
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
		for floor := numFloors -1; floor >= currFloor; floor-- {
			if assignedOrders[floor][elevio.BT_HallDown] == true {
				return floor;

			}
		}
	} else {
		for floor := currFloor; floor >= 0; floor-- {
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
		for floor := 0; floor <= currFloor; floor++ {
			if assignedOrders[floor][elevio.BT_HallUp] == true {
				return floor;
			}
		}
	}

	// Check other directions if no orders are found
	if currDir == Up {
		return calculateNextOrder(currFloor, Down, assignedOrders);
	}
	return calculateNextOrder(currFloor, Up, assignedOrders);
}

func hasOrders(assignedOrders [][] bool) (bool) {
	for floor := 0; floor < len(assignedOrders); floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			if assignedOrders[floor][orderType] == true {
				return true;
			}
		}
	}

	return false;
}

func calculateDirection(currFloor int, currOrder int) (OrderDir) {
	if currOrder > currFloor {
		return Up;
	}
	return Down;
}

func setOrder(buttonPress elevio.ButtonEvent, assignedOrders [][]bool, TurnOnLightsChanChan chan<- elevio.ButtonEvent) {
	assignedOrders[buttonPress.Floor][buttonPress.Button] = true;
	TurnOnLightsChanChan <- buttonPress;
}

func transmitState(localID NodeID, currState NodeBehaviour, currFloor int, currDir OrderDir, LocalNodeStateChanChan chan<- NodeState) {
	currNodeState := NodeState {
		ID: localID,
		Behaviour: currState,
		Floor: currFloor,
		Dir: currDir, 
	}

	LocalNodeStateChanChan <- currNodeState
}

// StateMachine ...
// GoRoutine for handling the states of a single elevator
func StateMachine(localID NodeID, numFloors int,
	NewOrderChanChan <-chan elevio.ButtonEvent,
	ArrivedAtFloorChanChan <-chan int,
	TurnOffLightsChanChan chan<- elevio.ButtonEvent,
	TurnOnLightsChanChan chan<- elevio.ButtonEvent,
	HallOrderChan chan<- [][] bool,
	CabOrderChan chan<- [] bool,
	LocallyAssignedOrdersChanChan <-chan [][] bool,
	CompletedOrderChanChan chan<- int,
	LocalNodeStateChanChan chan<- NodeState) {

	// Initialize variables	
	// -----
	doorOpenTime := 3 * time.Second;
	currOrder := -1;
	currFloor := -1;
	var currDir OrderDir = Up;
	doorTimer := time.NewTimer(0);

	assignedOrders := make([][]bool, numFloors);
	for i := range assignedOrders {
		assignedOrders[i] = make([]bool, 3);
	}

	for floor := range assignedOrders {
		for orderType := range assignedOrders[floor] {
			assignedOrders[floor][orderType] = false;
		}
	}

	// Initialize elevator
	// -----
	state := InitState
	nextState := state
	updateState := false;
	elevio.SetMotorDirection(elevio.MD_Up)
	elevio.SetDoorOpenLamp(false)
	fmt.Println("(fsm) Done initing");

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
			// Door has been open for the desired period of time
			if state != InitState {
				elevio.SetDoorOpenLamp(false)
				if hasOrders(assignedOrders) {
					nextState = MovingState
				} else {
					nextState = IdleState
				}
			}

		case a := <- LocallyAssignedOrdersChanChan:
			// Break if no changes!
			if reflect.DeepEqual(a, assignedOrders) {
				break;
			}

			assignedOrders = a

			if hasOrders(assignedOrders) && state != DoorOpenState {
				// Calculate new order
				nextState = MovingState
				// Change state from moving to moving
				updateState = true
			}

		case a := <- ArrivedAtFloorChanChan:
			currFloor = a;

			// Transmit state each when reached new floor
			transmitState(localID, state, currFloor, currDir, LocalNodeStateChanChan)

			if state == InitState {
				nextState = IdleState
			}

			if currFloor == currOrder {
				CompletedOrderChanChan <- currFloor
				//transmitHallOrdersChan(assignedOrders, HallOrderChan)
				//transmitCabOrdersChan(assignedOrders, CabOrderChan)
				nextState = DoorOpenState
			}
		}


		// State transition handling
		// -----
		if nextState != state || updateState {
			// Set new current state
			state = nextState
			switch state {
				case DoorOpenState:
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(doorOpenTime)

				case IdleState:
					elevio.SetMotorDirection(elevio.MD_Stop)

				case MovingState:
					currOrder = calculateNextOrder(currFloor, currDir, assignedOrders)

					// Already at desired floor
					if currOrder == currFloor {
						CompletedOrderChanChan <- currFloor
						nextState = DoorOpenState
						break
					}

					currDir = calculateDirection(currFloor, currOrder)

					// Set motor direction
					if currDir == Up {
						elevio.SetMotorDirection(elevio.MD_Up)
					} else {
						elevio.SetMotorDirection(elevio.MD_Down)
					}
			}
			// Transmit state each time state is changed
			transmitState(localID, state, currFloor, currDir, LocalNodeStateChanChan)
			updateState = false
		}
	}
}

