package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalAssigner"
	"./stateHandler"
	"fmt"
)

func main() {
	numFloors := 4;

	fsmChns := fsm.StateMachineChannels {
		NewOrder: make(chan elevio.ButtonEvent),
		ArrivedAtFloor: make(chan int),
	}
	iolightsChns := iolights.LightsChannels {
		TurnOnLights: make(chan elevio.ButtonEvent),
		TurnOffLights: make(chan elevio.ButtonEvent),
		FloorIndicator: make(chan int),
	}
	optimalAssignerChns := optimalAssigner.OptimalAssignerChannels {
		HallOrders: make(chan [][] bool),
		CabOrders: make(chan [] bool),
		ElevState: make(chan stateHandler.ElevState),
		AllElevStates: make(chan map[stateHandler.NodeID]stateHandler.ElevState),
	}
	stateHandlerChns := stateHandler.StateHandlerChannels {
		LocalElevStateChan: make(chan stateHandler.ElevState),
		RemoteElevStateChan: make(chan stateHandler.Elevstate),
		AllElevStatesChan: make(chan map[stateHandler.NodeID] statehandler.Elevstate),
	}


	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.FloorIndicator);
	go fsm.StateMachine(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, optimalAssignerChns.ElevState);
	go iolights.LightHandler(numFloors, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, iolightsChns.FloorIndicator);

	fmt.Println("Started all modules");
	go stateHandler.stateHandler(stateHandlerChns.LocalElevStateChan, stateHandlerChns.RemoteElevStateChan, stateHandlerChns.AllElevStatesChan)
	go optimalAssigner.Assigner(numFloors,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, optimalAssignerChns.ElevState, optimalAssignerChns.AllElevStates);

	for {};
}
