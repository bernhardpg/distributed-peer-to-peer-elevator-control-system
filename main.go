package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalAssigner"
	"fmt"
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
		CabOrdersChan: make(chan [] bool),
		ElevStateChan: make(chan fsm.ElevStateObject),
	}

	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.FloorIndicator);
	go fsm.StateMachine(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights,
		optimalAssignerChns.HallOrdersChan, optimalAssignerChns.CabOrdersChan, optimalAssignerChns.ElevStateChan);
	go iolights.LightHandler(numFloors, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, iolightsChns.FloorIndicator);

	fmt.Println("Started all modules");

	go optimalAssigner.Assigner(numFloors,
		optimalAssignerChns.HallOrdersChan, optimalAssignerChns.CabOrdersChan, optimalAssignerChns.ElevStateChan);

	for {};
}
