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
	}
	stateHandlerChns := stateHandler.StateHandlerChannels {
		LocalElevState: make(chan stateHandler.ElevState),
		RemoteElevState: make(chan stateHandler.ElevState),
		AllElevStates: make(chan map[stateHandler.NodeID] stateHandler.ElevState),
	}


	elevio.Init("localhost:15657", numFloors);

	go elevio.IOReader(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.FloorIndicator);
	go fsm.StateMachine(numFloors, fsmChns.NewOrder, fsmChns.ArrivedAtFloor, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, stateHandlerChns.LocalElevState);
	go iolights.LightHandler(numFloors, iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, iolightsChns.FloorIndicator);
	go stateHandler.StateHandler(stateHandlerChns.LocalElevState, stateHandlerChns.RemoteElevState, stateHandlerChns.AllElevStates)
	go optimalAssigner.Assigner(numFloors,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, stateHandlerChns.AllElevStates);

	fmt.Println("Started all modules");

	for {};
}
