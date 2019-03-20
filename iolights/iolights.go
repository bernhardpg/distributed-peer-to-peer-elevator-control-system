package iolights

import (
	"../elevio"
)

// LightsChannel ...
// Channels used for communication with the Elevator LightHandler
type LightsChannels struct {
	TurnOffLightsChan chan elevio.ButtonEvent
	TurnOnLightsChan chan elevio.ButtonEvent
	FloorIndicatorChan chan int
}

// LightHandler ...
// GoRoutine for controller the lights of a single elevator
func LightHandler(
	numFloors int,
	TurnOffLights <-chan elevio.ButtonEvent,
	TurnOnLights <-chan elevio.ButtonEvent,
	FloorIndicator <-chan int) {
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