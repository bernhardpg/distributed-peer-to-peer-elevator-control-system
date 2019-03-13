package fsm

import (
	"../elevio"
	"time"
)

type StateMachineChannels struct {
	NewOrder chan elevio.ButtonEvent
	ArrivedAtFloor chan int 
}

type ElevState int;
const (
	Init ElevState = iota;
	Idle
	DoorOpen
	Moving
)

type OrderDir int;
const (
	Up = iota;
	Down;
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

func calculateNextOrder(currFloor int, currDir OrderDir, assignedOrders [][] bool) (int) {
	numFloors := len(assignedOrders);

	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == Up {
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
	if currDir == Up {
		return calculateNextOrder(currFloor, Down, assignedOrders);
	} else {
		return calculateNextOrder(currFloor, Up, assignedOrders);
	}
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
	} else {
		return Down;
	}
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


func transitionTo(nextState ElevState, currFloor int, currDir OrderDir, assignedOrders [][] bool, doorTimer *time.Timer) (ElevState, int, OrderDir) {
	var state ElevState = nextState;
	var currOrder int = 0;
	var nextDir OrderDir = 0;

	switch nextState {
		case DoorOpen:
			elevio.SetMotorDirection(elevio.MD_Stop);
			elevio.SetDoorOpenLamp(true);
			doorTimer.Reset(3 * time.Second);

		case Idle:
			elevio.SetMotorDirection(elevio.MD_Stop);

		case Moving:
			currOrder = calculateNextOrder(currFloor, currDir, assignedOrders);
			nextDir = calculateDirection(currFloor, currOrder);

			if nextDir == Up {
				elevio.SetMotorDirection(elevio.MD_Up);
			} else {
				elevio.SetMotorDirection(elevio.MD_Down);
			}
	}

	return state, currOrder, nextDir;
}

func StateHandler(numFloors int, NewOrder chan elevio.ButtonEvent, ArrivedAtFloor chan int, TurnOffLights chan elevio.ButtonEvent, TurnOnLights chan elevio.ButtonEvent) {
	// Initialize variables	
	// -----
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

	var state ElevState = Init;

	// Initialize elevator
	// -----
	elevio.SetMotorDirection(elevio.MD_Up);

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
			// Door has been open for the desired period of time
			if state != Init {
				elevio.SetDoorOpenLamp(false);
				if hasOrders(assignedOrders) {
					state, currOrder, currDir = transitionTo(Moving, currFloor, currDir, assignedOrders, doorTimer);
				} else {
					state,_,_ = transitionTo(Idle, currFloor, currDir, assignedOrders, doorTimer);
				}
			}

		case a := <- ArrivedAtFloor:
			currFloor = a;

			if state == Init {
				state,_,_ = transitionTo(Idle, currFloor, currDir, assignedOrders, doorTimer);
			}

			if currFloor == currOrder {
				clearOrdersAtFloor(currFloor, assignedOrders, TurnOffLights);
				state,_,_ = transitionTo(DoorOpen, currFloor, currDir, assignedOrders, doorTimer);
			}

		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			
			// Only open door if already on floor (and not moving)
			if a.Floor == currFloor && state != Moving {
				// Open door without calculating new order
				state,_,_ = transitionTo(DoorOpen, currFloor, currDir, assignedOrders, doorTimer);
			} else {
				setOrder(a, assignedOrders, TurnOnLights);
				if state != DoorOpen {
					// Calculate new order
					state, currOrder, currDir = transitionTo(Moving, currFloor, currDir, assignedOrders, doorTimer);
				}
			}
		}
	}
}
