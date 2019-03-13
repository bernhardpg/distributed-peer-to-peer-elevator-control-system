package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalAssigner"
)

func main() {
	numFloors := 4;

	fsmChns := fsm.StateMachineChannels {
		NewOrder: make(chan elevio.ButtonEvent),
		ArrivedAtFloor: make(chan int),
	}
	iolightsChns := iolights.LightsChannel {
		TurnOnLights: make(chan elevio.ButtonEvent),
		TurnOffLights: make(chan elevio.ButtonEvent),
		FloorIndicator: make(chan int),
	}
	optimalAssignerChns := optimalAssigner.OptimalAssignerChns {
		HallOrdersChan: make(chan [][] bool),
	}

	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.FloorIndicator);
	go fsm.StateHandler(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, optimalAssignerChns.HallOrdersChan);
	go iolights.LightHandler(numFloors, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, iolightsChns.FloorIndicator);

	go optimalAssigner.Assigner(optimalAssignerChns.HallOrdersChan);

	for {};
}
