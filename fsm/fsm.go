package fsm

import (
	"../driver/elevio"
	"fmt"
)

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

type ElevState int;
const (
	Init ElevState = iota;
	Idle
	DoorOpen
	Moving
)

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
}

func FSM() {
	numFloors := 4;
	numStates := 3; // Will always be 3
	elevio.Init("localhost:15657", numFloors);

	// Initialize locally assigned orders matrix
	var locallyAssignedOrders = make([][]Req, numFloors);
	for i := range assignedOrders {
		assignedOrders[i] = make([]Req, numStates);
	}

	// Initialize all to unknown
	for floor := range assignedOrders {
		for orderType := range assignedOrders[floor] {
			assignedOrders[floor][orderType].state = Unknown;
		}
	}

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	// INITALIZE ELEVATOR
	var currState ElevState = Init;
	var d elevio.MotorDirection = elevio.MD_Up
	elevio.SetMotorDirection(d);

	for {
		select {
		case a := <-drv_buttons:
			fmt.Printf("%+v\n", a);
			elevio.SetButtonLamp(a.Button, a.Floor, true);

		case a := <-drv_floors:
			if currState == Init {
				elevio.SetMotorDirection(elevio.MD_Stop);
			} else {
				fmt.Printf("%+v\n", a)
				if a == numFloors-1 {
					d = elevio.MD_Down
				} else if a == 0 {
					d = elevio.MD_Up
				}
			}
			elevio.SetMotorDirection(d)

		case a := <-drv_obstr:
			fmt.Printf("%+v\n", a);
			if a {
				elevio.SetMotorDirection(elevio.MD_Stop);
			} else {
				elevio.SetMotorDirection(d);
			}

		case a := <-drv_stop:
			fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := elevio.ButtonType(0); b < 3; b++ {
					elevio.SetButtonLamp(b, f, false);
				}
			}
		}
	}
}
