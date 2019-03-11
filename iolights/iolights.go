package iolights

import (
	"../elevio"
)

type LightsChannel struct {
	TurnOffLights chan elevio.ButtonEvent
	TurnOnLights chan elevio.ButtonEvent
}

func LightHandler(TurnOffLights chan elevio.ButtonEvent, TurnOnLights chan elevio.ButtonEvent) {
	for {
		select {
		case a := <- TurnOffLights:
			elevio.SetButtonLamp(a.Button, a.Floor, false);
		case a := <- TurnOnLights:
			elevio.SetButtonLamp(a.Button, a.Floor, true);
		}
	}
}