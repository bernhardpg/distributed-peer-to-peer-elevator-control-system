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
		for floor := currFloor + 1; floor <= numFloors - 1; floor++ {
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
		for floor := currFloor - 1; floor >= 0; floor-- {
			if assignedOrders[floor][elevio.BT_HallUp] == true {
				return floor;
			}
		}
	}

	// Check other directions if no orders are found
	if currDir == Down {
		return calculateNextOrder(currFloor, Up, assignedOrders);
	} else {
		return calculateNextOrder(currFloor, Down, assignedOrders);
	}
}

func hasOrders(assignedOrders [][]bool) (bool) {
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

	doorOpen := false;
	moving := false;
	state := make(chan ElevState);

	// Initialize elevator
	// -----
	elevio.SetMotorDirection(elevio.MD_Up);
	initialized := false;

	// State transition handler
	// -----
	go func() {
		doorOpenTime := 3 * time.Second;

		for {
			select {
			case a := <- state:
				switch a {

				case DoorOpen:
					moving = false;
					doorOpen = true;
					elevio.SetMotorDirection(elevio.MD_Stop);
					elevio.SetDoorOpenLamp(true);
					doorTimer.Reset(doorOpenTime);

				case Idle:
					elevio.SetMotorDirection(elevio.MD_Stop);

				case Moving:
					moving = true;
					currOrder = calculateNextOrder(currFloor, currDir, assignedOrders);
					currDir = calculateDirection(currFloor, currOrder);

					if currDir == Up {
						elevio.SetMotorDirection(elevio.MD_Up);
					} else {
						elevio.SetMotorDirection(elevio.MD_Down);
					}
				}
			}	
		}
	}()

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
		// Door have been open for the desired period of time
			doorOpen = false;
			if initialized {
				elevio.SetDoorOpenLamp(false);
				if hasOrders(assignedOrders) {
					state <- Moving;
				} else {
					state <- Idle;
				}
			}

		case a := <- ArrivedAtFloor:
			currFloor = a;

			if !initialized {
				initialized = true;
				state <- Idle;
			}

			if currFloor == currOrder {
				clearOrdersAtFloor(currFloor, assignedOrders, TurnOffLights);
				state <- DoorOpen;

			}


		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			
			// Only open door if already on floor (and not moving)
			if a.Floor == currFloor && !moving {
				state <- DoorOpen;
			} else {
				setOrder(a, assignedOrders, TurnOnLights);
				if !doorOpen {
					state <- Moving;
				}
			}
		}
	}
}
