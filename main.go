package main

import (
	"./fsm"
	"./driver/elevio"
)

func main() {
	numFloors := 4;

	fsmChans := fsm.StateMachineChannels {
		NewOrder: make(chan elevio.ButtonEvent),
		ArrivedAtFloor: make(chan int),
	}

	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChans.NewOrder, fsmChans.ArrivedAtFloor);
	go fsm.StateHandler(numFloors, fsmChans.NewOrder, fsmChans.ArrivedAtFloor);

	for {};
}
