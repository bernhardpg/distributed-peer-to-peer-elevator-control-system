package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalAssigner"
	"./nodeStatesHandler"
	"fmt"
)


func main() {
	var localID fsm.NodeID = 1;
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
	nodeStatesHandlerChns := nodeStatesHandler.NodeStatesHandlerChannels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan fsm.NodeState),
		AllNodeStatesChan: make(chan map[fsm.NodeID] fsm.NodeState, 2),
		// TODO: does allNodeStates need a buffer bigger than 1?
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
		nodeStatesHandlerChns.LocalNodeStateChan);

	go iolights.LightHandler(numFloors,
		iolightsChns.TurnOffLightsChan, iolightsChns.TurnOnLightsChan, iolightsChns.FloorIndicatorChan);

	go nodeStatesHandler.NodeStatesHandler(localID,
		nodeStatesHandlerChns.LocalNodeStateChan, nodeStatesHandlerChns.RemoteNodeStatesChan, nodeStatesHandlerChns.AllNodeStatesChan)

	go optimalAssigner.Assigner(localID, numFloors,
		optimalAssignerChns.HallOrdersChan, optimalAssignerChns.CabOrdersChan, optimalAssignerChns.LocallyAssignedOrdersChan, optimalAssignerChns.NewOrderChan, optimalAssignerChns.CompletedOrderChan,
		nodeStatesHandlerChns.AllNodeStatesChan); 

	fmt.Println("(main) Started all modules");

	for {};
}
