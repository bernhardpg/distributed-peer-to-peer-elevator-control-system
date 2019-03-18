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
	var localID stateHandler.NodeID = 1;
	numFloors := 4;

	fsmChns := fsm.StateMachineChannels {
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
		NewOrder: make(chan elevio.ButtonEvent),  // TODO move to consensus module
		CompletedOrder: make(chan int),
		LocallyAssignedOrders: make(chan [][] bool),
	}
	stateHandlerChns := stateHandler.StateHandlerChannels {
		LocalElevState: make(chan stateHandler.ElevState),
		RemoteElevState: make(chan stateHandler.ElevState),
		AllElevStates: make(chan map[stateHandler.NodeID] stateHandler.ElevState),
	}


	elevio.Init("localhost:15657", numFloors);

	// Start modules
	// -----
	go elevio.IOReader(numFloors,
		optimalAssignerChns.NewOrder, fsmChns.ArrivedAtFloor,
		iolightsChns.FloorIndicator);

	go fsm.StateMachine(localID, numFloors,
		optimalAssignerChns.NewOrder, fsmChns.ArrivedAtFloor,
		iolightsChns.TurnOffLights, iolightsChns.TurnOnLights,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, optimalAssignerChns.LocallyAssignedOrders, optimalAssignerChns.CompletedOrder,
		stateHandlerChns.LocalElevState);

	go iolights.LightHandler(numFloors,
		iolightsChns.TurnOffLights, iolightsChns.TurnOnLights, iolightsChns.FloorIndicator);

	go stateHandler.StateHandler(localID,
		stateHandlerChns.LocalElevState, stateHandlerChns.RemoteElevState, stateHandlerChns.AllElevStates)

	go optimalAssigner.Assigner(localID, numFloors,
		optimalAssignerChns.HallOrders, optimalAssignerChns.CabOrders, optimalAssignerChns.LocallyAssignedOrders, optimalAssignerChns.NewOrder, optimalAssignerChns.CompletedOrder,
		stateHandlerChns.AllElevStates); 

	fmt.Println("Started all modules");

	for {};
}
