package fsm

import (
	"../elevio"
	"time"
	"../stateHandler"
	"fmt"
	"reflect"
)

// StateMachineChannels ...
// Channels used for communication with the Elevator FSM
type StateMachineChannels struct {
	NewOrderChan chan elevio.ButtonEvent
	ArrivedAtFloorChan chan int 
}

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

func calculateNextOrder(currFloor int, currDir stateHandler.OrderDir, assignedOrders [][] bool) (int) {
	numFloors := len(assignedOrders);
	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == stateHandler.Up {
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
	if currDir == stateHandler.Up {
		return calculateNextOrder(currFloor, stateHandler.Down, assignedOrders);
	}
	return calculateNextOrder(currFloor, stateHandler.Up, assignedOrders);
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

func calculateDirection(currFloor int, currOrder int) (stateHandler.OrderDir) {
	if currOrder > currFloor {
		return stateHandler.Up;
	}
	return stateHandler.Down;
}

func setOrder(buttonPress elevio.ButtonEvent, assignedOrders [][]bool, TurnOnLightsChanChan chan<- elevio.ButtonEvent) {
	assignedOrders[buttonPress.Floor][buttonPress.Button] = true;
	TurnOnLightsChanChan <- buttonPress;
}

func transmitState(localID stateHandler.NodeID, currState stateHandler.BehaviourState, currFloor int, currDir stateHandler.OrderDir, LocalElevStateChanChan chan<- stateHandler.ElevState) {
	currElevState := stateHandler.ElevState {
		ID: localID,
		State: currState,
		Floor: currFloor,
		Dir: currDir, 
	}

	LocalElevStateChanChan <- currElevState
}

// StateMachine ...
// GoRoutine for handling the states of a single elevator
func StateMachine(localID stateHandler.NodeID, numFloors int,
	NewOrderChanChan <-chan elevio.ButtonEvent,
	ArrivedAtFloorChanChan <-chan int,
	TurnOffLightsChanChan chan<- elevio.ButtonEvent,
	TurnOnLightsChanChan chan<- elevio.ButtonEvent,
	HallOrderChan chan<- [][] bool,
	CabOrderChan chan<- [] bool,
	LocallyAssignedOrdersChanChan <-chan [][] bool,
	CompletedOrderChanChan chan<- int,
	LocalElevStateChanChan chan<- stateHandler.ElevState) {

	// Initialize variables	
	// -----
	doorOpenTime := 3 * time.Second;
	currOrder := -1;
	currFloor := -1;
	var currDir stateHandler.OrderDir = stateHandler.Up;
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
	state := stateHandler.InitState
	nextState := state
	updateState := false;
	elevio.SetMotorDirection(elevio.MD_Up)
	elevio.SetDoorOpenLamp(false)
	fmt.Println("Done initing fsm");

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
			// Door has been open for the desired period of time
			if state != stateHandler.InitState {
				elevio.SetDoorOpenLamp(false)
				if hasOrders(assignedOrders) {
					nextState = stateHandler.Moving
				} else {
					nextState = stateHandler.Idle
				}
			}

		case a := <- LocallyAssignedOrdersChanChan:
			// Break if no changes!
			if reflect.DeepEqual(a, assignedOrders) {
				break;
			}

			assignedOrders = a

			if hasOrders(assignedOrders) && state != stateHandler.DoorOpen {
				// Calculate new order
				nextState = stateHandler.Moving
				// Change state from moving to moving
				updateState = true
			}

		case a := <- ArrivedAtFloorChanChan:
			currFloor = a;

			// Transmit state each when reached new floor
			transmitState(localID, state, currFloor, currDir, LocalElevStateChanChan)

			if state == stateHandler.InitState {
				nextState = stateHandler.Idle
			}

			if currFloor == currOrder {
				CompletedOrderChanChan <- currFloor
				//transmitHallOrdersChan(assignedOrders, HallOrderChan)
				//transmitCabOrdersChan(assignedOrders, CabOrderChan)
				nextState = stateHandler.DoorOpen
			}
		}


		// State transition handling
		// -----
		if nextState != state || updateState {
			// Set new current state
			state = nextState
			switch state {
				case stateHandler.DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(doorOpenTime)

				case stateHandler.Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)

				case stateHandler.Moving:
					currOrder = calculateNextOrder(currFloor, currDir, assignedOrders)

					// Already at desired floor
					if currOrder == currFloor {
						CompletedOrderChanChan <- currFloor
						nextState = stateHandler.DoorOpen
						break
					}

					currDir = calculateDirection(currFloor, currOrder)

					// Set motor direction
					if currDir == stateHandler.Up {
						elevio.SetMotorDirection(elevio.MD_Up)
					} else {
						elevio.SetMotorDirection(elevio.MD_Down)
					}
			}
			// Transmit state each time state is changed
			transmitState(localID, state, currFloor, currDir, LocalElevStateChanChan)
			updateState = false
		}
	}
}

