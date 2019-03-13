package fsm

import (
	"../elevio"
	"time"
)

// StateMachineChannels ...
// Channels used for communication with the Elevator FSM
type StateMachineChannels struct {
	NewOrder chan elevio.ButtonEvent
	ArrivedAtFloor chan int 
}

type elevState int;
const (
	initState elevState = iota;
	idle
	doorOpen
	moving
)

type orderDir int;
const (
	up = iota;
	down;
)

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

func calculateNextOrder(currFloor int, currDir orderDir, assignedOrders [][] bool) (int) {
	numFloors := len(assignedOrders);

	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == up {
		for floor := currFloor + 1; floor <= numFloors - 1; floor++ {
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
		for floor := numFloors -1 ; floor >= currFloor + 1; floor-- {
			if assignedOrders[floor][elevio.BT_HallDown] == true {
				return floor;

			}
		}
	} else {
		for floor := currFloor - 1; floor >= 0; floor-- {
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
		for floor := 0; floor <= currFloor - 1; floor++ {
			if assignedOrders[floor][elevio.BT_HallUp] == true {
				return floor;
			}
		}
	}

	// Check other directions if no orders are found
	if currDir == up {
		return calculateNextOrder(currFloor, down, assignedOrders);
	}
	return calculateNextOrder(currFloor, up, assignedOrders);
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

func calculateDirection(currFloor int, currOrder int) (orderDir) {
	if currOrder > currFloor {
		return up;
	}
	return down;
}

func clearOrdersAtFloor(currFloor int, assignedOrders [][]bool, TurnOffLights chan elevio.ButtonEvent) {
	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
		assignedOrders[currFloor][orderType] = false;
		TurnOffLights <- elevio.ButtonEvent{currFloor, elevio.ButtonType(orderType)};
	}
}

func setOrder(buttonPress elevio.ButtonEvent, assignedOrders [][]bool, TurnOnLights chan elevio.ButtonEvent) {
	assignedOrders[buttonPress.Floor][buttonPress.Button] = true;
	TurnOnLights <- buttonPress;
}


func transitionTo(nextState elevState, currFloor int, currDir orderDir, assignedOrders [][] bool, doorTimer *time.Timer) (elevState, int, orderDir) {
	state := nextState;
	currOrder := 0;
	var nextDir orderDir = up;

	switch nextState {
		case doorOpen:
			elevio.SetMotorDirection(elevio.MD_Stop);
			elevio.SetDoorOpenLamp(true);
			doorTimer.Reset(3 * time.Second);

		case idle:
			elevio.SetMotorDirection(elevio.MD_Stop);

		case moving:
			currOrder = calculateNextOrder(currFloor, currDir, assignedOrders);
			nextDir = calculateDirection(currFloor, currOrder);

			if nextDir == up {
				elevio.SetMotorDirection(elevio.MD_Up);
			} else {
				elevio.SetMotorDirection(elevio.MD_Down);
			}
	}

	return state, currOrder, nextDir;
}


// StateHandler ...
// GoRoutine for handling the states of a single elevator
func StateHandler(numFloors int, NewOrder chan elevio.ButtonEvent, ArrivedAtFloor chan int, TurnOffLights chan elevio.ButtonEvent, TurnOnLights chan elevio.ButtonEvent) {
	// Initialize variables	
	// -----
	currOrder := -1;
	currFloor := -1;
	var currDir orderDir = up;
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

	var state elevState = initState;

	// Initialize elevator
	// -----
	elevio.SetMotorDirection(elevio.MD_Up);

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
			// Door has been open for the desired period of time
			if state != initState {
				elevio.SetDoorOpenLamp(false);
				if hasOrders(assignedOrders) {
					state, currOrder, currDir = transitionTo(moving, currFloor, currDir, assignedOrders, doorTimer);
				} else {
					state,_,_ = transitionTo(idle, currFloor, currDir, assignedOrders, doorTimer);
				}
			}

		case a := <- ArrivedAtFloor:
			currFloor = a;

			if state == initState {
				state,_,_ = transitionTo(idle, currFloor, currDir, assignedOrders, doorTimer);
			}

			if currFloor == currOrder {
				clearOrdersAtFloor(currFloor, assignedOrders, TurnOffLights);
				state,_,_ = transitionTo(doorOpen, currFloor, currDir, assignedOrders, doorTimer);
			}

		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			
			// Only open door if already on floor (and not moving)
			if a.Floor == currFloor && state != moving {
				// Open door without calculating new order
				state,_,_ = transitionTo(doorOpen, currFloor, currDir, assignedOrders, doorTimer);
			} else {
				setOrder(a, assignedOrders, TurnOnLights);
				if state != doorOpen {
					// Calculate new order
					state, currOrder, currDir = transitionTo(moving, currFloor, currDir, assignedOrders, doorTimer);
				}
			}
		}
	}
}
