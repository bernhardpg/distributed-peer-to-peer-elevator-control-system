package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalOrderAssigner"
	"./nodeStatesHandler"
	"./network"
	"fmt"
	"flag"
	"strconv"
)


func main() {
	
	numFloors := 4;

	// ID Handling
	// ------

	// Pass the ID in the command line with `go run main.go -id=our_id`
	IDptr := flag.Int("id", 1, "LocalID of the node");
	// Pass the port number in the command line with `go run main.go -port=our_id`
	portPtr := flag.Int("port", 15657, "Port for connecting to elevator");

	flag.Parse()

	localID := (network.NodeID)(*IDptr)
	port := *portPtr

	fmt.Println("(main) localID:", localID)
	fmt.Println("(main) port:", port)

	// Connect to elevator through tcp (either hardware or simulator)
	// -----
	elevio.Init("localhost:" + strconv.Itoa(port), numFloors);


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
		AllNodeStatesChan: make(chan map[network.NodeID] fsm.NodeState),
	}
	networkChns := network.Channels {
		LocalNodeStateChan: make(chan fsm.NodeState),
	}



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
		localID,
		networkChns.LocalNodeStateChan)

	fmt.Println("(main) Started all modules");

	for {
		select {}
	}
}
