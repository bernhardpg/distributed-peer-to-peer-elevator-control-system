package iolights

import (
	"../elevio"
)

type LightsChannel struct {
	TurnOffLights chan elevio.ButtonEvent
	TurnOnLights chan elevio.ButtonEvent
	FloorIndicator chan int
}

func LightHandler(numFloors int, TurnOffLights chan elevio.ButtonEvent, TurnOnLights chan elevio.ButtonEvent, FloorIndicator chan int) {
	// Turn off all lights at init
	for floor := 0; floor < numFloors; floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			elevio.SetButtonLamp(orderType, floor, false);
		}
	}


	for {
		select {
		case a := <- TurnOffLights:
			elevio.SetButtonLamp(a.Button, a.Floor, false);
		case a := <- TurnOnLights:
			elevio.SetButtonLamp(a.Button, a.Floor, true);
		case a := <- FloorIndicator:
			elevio.SetFloorIndicator(a);
		}

	}
}