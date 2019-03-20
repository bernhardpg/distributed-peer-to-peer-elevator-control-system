package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalOrderAssigner"
	"./nodeStatesHandler"
	"./network"
	"fmt"
)


func main() {
	var localID network.NodeID = 1;
	numFloors := 4;

	// Init channels
	// -----
	fsmChns := fsm.StateMachineChannels {
		ArrivedAtFloorChan: make(chan int),
	}
	iolightsChns := iolights.LightsChannels {
		TurnOnLightsChan: make(chan elevio.ButtonEvent),
		TurnOffLightsChan: make(chan elevio.ButtonEvent),
		FloorIndicatorChan: make(chan int),
	}
	optimalOrderAssignerChns := optimalOrderAssigner.OptimalOrderAssignerChannels {
		NewOrderChan: make(chan elevio.ButtonEvent), // TODO move to consensus module
		CompletedOrderChan: make(chan int),
		LocallyAssignedOrdersChan: make(chan [][] bool, 2),
		// Needs a buffer size bigger than one because the optimalOrderAssigner might send on this channel multiple times before FSM manages to receive!
	}
	nodeStatesHandlerChns := nodeStatesHandler.NodeStatesHandlerChannels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan fsm.NodeState),
		AllNodeStatesChan: make(chan map[network.NodeID] fsm.NodeState),
	}
	networkChns := network.Channels {
		LocalNodeStateChan: make(chan fsm.NodeState),
	}


	elevio.Init("localhost:15657", numFloors);

	// Start modules
	// -----
	go elevio.IOReader(
		numFloors,
		optimalOrderAssignerChns.NewOrderChan,
		fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan)

	go fsm.StateMachine(
		numFloors,
		fsmChns.ArrivedAtFloorChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		optimalOrderAssignerChns.CompletedOrderChan,
		nodeStatesHandlerChns.LocalNodeStateChan)

	go iolights.LightHandler(
		numFloors,
		iolightsChns.TurnOffLightsChan,
		iolightsChns.TurnOnLightsChan,
		iolightsChns.FloorIndicatorChan)

	go nodeStatesHandler.NodeStatesHandler(
		localID,
		nodeStatesHandlerChns.LocalNodeStateChan,
		nodeStatesHandlerChns.RemoteNodeStatesChan,
		nodeStatesHandlerChns.AllNodeStatesChan,
		networkChns.LocalNodeStateChan)

	go optimalOrderAssigner.Assigner(
		localID, numFloors,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		optimalOrderAssignerChns.NewOrderChan,
		optimalOrderAssignerChns.CompletedOrderChan,
		nodeStatesHandlerChns.AllNodeStatesChan,
		iolightsChns.TurnOffLightsChan,
		iolightsChns.TurnOnLightsChan)

	go network.Module(
		networkChns.LocalNodeStateChan)

	fmt.Println("(main) Started all modules");

	for {
		select {}
	}
}
