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
			// Check orders of opposite directon last
			for floor := currFloor - 1; floor >= 0; floor-- {
				if assignedOrders[floor][elevio.BT_HallUp] == true {
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

func clearOrdersAtFloor(currFloor int, assignedOrders [][]bool) {
	for i := 0; i < 3; i++ {
		assignedOrders[currFloor][i] = false;
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
	doorOpen := false;
	moving := false;
	elevio.SetMotorDirection(elevio.MD_Up);
	
	state := make(chan ElevState);

	// State transition handler
	go func() {
		doorOpenTime := 3 * time.Second;

		for {
			select {
			case a := <- state:
				if a == DoorOpen {
					moving = false;
					doorOpen = true;
					elevio.SetDoorOpenLamp(true);
					doorTimer.Reset(doorOpenTime);
				} else if a == Idle {
					elevio.SetMotorDirection(elevio.MD_Stop);
				} else if a == Moving {
					moving = true;
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
	}()


	for {
		select {
		// Doors have been open for desired period of time
		case <- doorTimer.C:
			doorOpen = false;
			if initialized {
				elevio.SetDoorOpenLamp(false);
				if hasOrders(assignedOrders) {
					fmt.Println("More orders, continue");
					state <- Moving;
				} else {
					fmt.Println("No orders, go idle");
					state <- Idle;
				}
			}

		case a := <- ArrivedAtFloor:
			currFloor = a;
			fmt.Println("Arrived at floor: ", currFloor);

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
				clearOrdersAtFloor(currFloor, assignedOrders);
				elevio.SetMotorDirection(elevio.MD_Stop);
				state <- DoorOpen;				
			}


		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			
			// Only open door if already on floor
			if a.Floor == currFloor && !moving{
				state <- DoorOpen;
			} else {
				assignedOrders[a.Floor][a.Button] = true;
				if !doorOpen {
					state <- Moving;
				}
			}
		}
	}
}
