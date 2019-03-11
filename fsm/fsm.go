package fsm

import (
	"fmt"
	"../driver/elevio"
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

// ------

	
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


//if (!initialized)
//	don't accept orders until initialized

//STATES: Init, Idle, DoorOpen, Moving

/*func goIdle() {
	if (!moving && NoOrders) || (doorOpen && time == 3 sec && NoOrders) || (lastState == init) {
		idle = true;
	}
	else {
		idle = false;
	}
}*/

func calculateNextOrder(currFloor int, currDir OrderDir, assignedOrders [][] bool) (int) {
	if (!hasOrders(assignedOrders)) {
		fmt.Println("No orders! Will loop forever");
		return -1;
	}

	// Return first order in current direction
	if currDir == Up {
		for floor := 0; floor < currFloor; floor++ {
			for orderType := 0; orderType < 3; orderType++ {
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
			}
		}
	} else {
		for floor := currFloor - 1; floor >= 0; floor-- {
			for orderType := 0; orderType < 3; orderType++ {
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
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
		for orderType := 0; orderType < 3; orderType++ {
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

func transitionHandler(state chan ElevState, currFloor int, currOrder int, currDir OrderDir, assignedOrders [][]bool, doorTimer *time.Timer) {
	doorOpenTime := 3 * time.Second;

	for {
		select {
		case a := <- state:
			fmt.Println("Received state msg: ", a);
			if a == DoorOpen {
				elevio.SetDoorOpenLamp(true);
				doorTimer.Reset(doorOpenTime);
			} else if a == Idle {
				elevio.SetMotorDirection(elevio.MD_Stop);
			} else if a == Moving {
				currOrder = calculateNextOrder(currFloor, currDir, assignedOrders);
				fmt.Println("New current order: ", currOrder);
				currDir = calculateDirection(currFloor, currOrder);

				if currDir == Up {
					elevio.SetMotorDirection(elevio.MD_Up);
				} else {
					elevio.SetMotorDirection(elevio.MD_Down);
				}
			}
		}	
	}
}

func FSM(numFloors int, NewOrder chan elevio.ButtonEvent, ArrivedAtFloor chan int) {	
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

	initialized := false;
	elevio.SetMotorDirection(elevio.MD_Up);
	
	state := make(chan ElevState);
	go transitionHandler(state, currFloor, currOrder, currDir, assignedOrders, doorTimer);

	for {
		select {
		// Doors have been open for desired period of time
		case <- doorTimer.C:
			if initialized {
				elevio.SetDoorOpenLamp(false);
				if hasOrders(assignedOrders) {
					state <- Moving;
				} else {
					state <- Idle;
				}
			}

		case a := <- ArrivedAtFloor:
			fmt.Println("Arrived at floor: ", a);
			currFloor = a;

			if !initialized {
				initialized = true;
				fmt.Println("Going to idle state");
				state <- Idle;
			}

			// TODO Do we need to check for top and bottom?
			/*// Hit the top floor
			if currFloor == numFloors - 1 {

			}*/

			if currFloor == currOrder {
				fmt.Println("Arrived at desired floor");
				elevio.SetMotorDirection(elevio.MD_Stop);
				state <- DoorOpen;				
			}


		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			assignedOrders[a.Floor][a.Button] = true;
			state <- Moving;
			// TODO Keep door open when ordered to current floor
		}
	}
}
