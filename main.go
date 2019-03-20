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
		ArrivedAtFloorChan: make(chan int),
	}
	iolightsChns := iolights.LightsChannels {
		TurnOnLightsChan: make(chan elevio.ButtonEvent),
		TurnOffLightsChan: make(chan elevio.ButtonEvent),
		FloorIndicatorChan: make(chan int),
	}
	optimalAssignerChns := optimalAssigner.OptimalAssignerChannels {
		HallOrdersChan: make(chan [][] bool),
		CabOrdersChan: make(chan [] bool),
		NewOrderChan: make(chan elevio.ButtonEvent),  // TODO move to consensus module
		CompletedOrderChan: make(chan int),
		LocallyAssignedOrdersChan: make(chan [][] bool, 2),
		// Needs a buffer size bigger than one because the optimalAssigner might send on this channel multiple times before FSM manages to receive!
	}
	stateHandlerChns := stateHandler.StateHandlerChannels {
		LocalElevStateChan: make(chan stateHandler.ElevState),
		RemoteElevStateChan: make(chan stateHandler.ElevState),
		AllElevStatesChan: make(chan map[stateHandler.NodeID] stateHandler.ElevState, 2), // TODO: does allElevStates need a buffer bigger than 1?
	}


	elevio.Init("localhost:15657", numFloors);

	// Start modules
	// -----
	go elevio.IOReader(numFloors,
		optimalAssignerChns.NewOrderChan, fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan);

	go fsm.StateMachine(localID, numFloors,
		optimalAssignerChns.NewOrderChan, fsmChns.ArrivedAtFloorChan,
		iolightsChns.TurnOffLightsChan, iolightsChns.TurnOnLightsChan,
		optimalAssignerChns.HallOrdersChan, optimalAssignerChns.CabOrdersChan, optimalAssignerChns.LocallyAssignedOrdersChan, optimalAssignerChns.CompletedOrderChan,
		stateHandlerChns.LocalElevStateChan);

	go iolights.LightHandler(numFloors,
		iolightsChns.TurnOffLightsChan, iolightsChns.TurnOnLightsChan, iolightsChns.FloorIndicatorChan);

	go stateHandler.StateHandler(localID,
		stateHandlerChns.LocalElevStateChan, stateHandlerChns.RemoteElevStateChan, stateHandlerChns.AllElevStatesChan)

	go optimalAssigner.Assigner(localID, numFloors,
		optimalAssignerChns.HallOrdersChan, optimalAssignerChns.CabOrdersChan, optimalAssignerChns.LocallyAssignedOrdersChan, optimalAssignerChns.NewOrderChan, optimalAssignerChns.CompletedOrderChan,
		stateHandlerChns.AllElevStatesChan); 

	fmt.Println("Started all modules");

	for {};
}
