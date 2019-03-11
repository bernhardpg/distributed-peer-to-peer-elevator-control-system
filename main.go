package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
)

func main() {
	numFloors := 4;

	fsmChans := fsm.StateMachineChannels {
		NewOrder: make(chan elevio.ButtonEvent),
		ArrivedAtFloor: make(chan int),
	}
	iolightsChans := iolights.LightsChannel {
		TurnOnLights: make(chan elevio.ButtonEvent),
		TurnOffLights: make(chan elevio.ButtonEvent),
	}

	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChans.NewOrder, fsmChans.ArrivedAtFloor);
	go fsm.StateHandler(numFloors, fsmChans.NewOrder, fsmChans.ArrivedAtFloor, iolightsChans.TurnOffLights, iolightsChans.TurnOnLights);
	go iolights.LightHandler(iolightsChans.TurnOffLights, iolightsChans.TurnOnLights);

	for {};
}
