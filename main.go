package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalOrderAssigner"
	"./nodeStatesHandler"
	"./network"
	"./consensusModule/hallConsensus"
	"./consensusModule/cabConsensus"
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
		TurnOffHallLightChan: make(chan elevio.ButtonEvent),
		TurnOnHallLightChan: make(chan elevio.ButtonEvent),
		TurnOffCabLightChan: make(chan elevio.ButtonEvent),
		TurnOnCabLightChan: make(chan elevio.ButtonEvent),
	}
	optimalOrderAssignerChns := optimalOrderAssigner.OptimalOrderAssignerChannels {
		NewOrderChan: make(chan elevio.ButtonEvent), // TODO move to consensus module
		CompletedOrderChan: make(chan int),
		LocallyAssignedOrdersChan: make(chan [][] bool, 2),
		// Needs a buffer size bigger than one because the optimalOrderAssigner might send on this channel multiple times before FSM manages to receive!
	}
	nodeStatesHandlerChns := nodeStatesHandler.NodeStatesHandlerChannels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan network.NodeStateMsg, 2),
		AllNodeStatesChan: make(chan map[network.NodeID] fsm.NodeState, 2),
		NodeLostChan: make(chan network.NodeID),
	}
	networkChns := network.Channels {
		LocalNodeStateChan: make(chan fsm.NodeState),
	}
	hallConsensusChns := hallConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan elevio.ButtonEvent),
		ConfirmedOrdersChan: make(chan [][] bool),
	}
	cabConsensusChns := cabConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan int),
	}

	// TODO Double check channel buffering!


	// Start modules
	// -----
	go elevio.IOReader(
		numFloors,
		hallConsensusChns.NewOrderChan,
		cabConsensusChns.NewOrderChan,
		fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan,
		optimalOrderAssignerChns.NewOrderChan)

	go fsm.StateMachine(
		numFloors,
		fsmChns.ArrivedAtFloorChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		cabConsensusChns.CompletedOrderChan,
		nodeStatesHandlerChns.LocalNodeStateChan,
		optimalOrderAssignerChns.CompletedOrderChan)

	go iolights.LightHandler(
		numFloors,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		iolightsChns.TurnOffCabLightChan,
		iolightsChns.TurnOnCabLightChan,
		iolightsChns.FloorIndicatorChan)

	go nodeStatesHandler.NodeStatesHandler(
		localID,
		nodeStatesHandlerChns.LocalNodeStateChan,
		nodeStatesHandlerChns.RemoteNodeStatesChan,
		nodeStatesHandlerChns.AllNodeStatesChan,
		nodeStatesHandlerChns.NodeLostChan,
		networkChns.LocalNodeStateChan)

	go optimalOrderAssigner.Assigner(
		localID,
		numFloors,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		optimalOrderAssignerChns.NewOrderChan,
		optimalOrderAssignerChns.CompletedOrderChan,
		nodeStatesHandlerChns.AllNodeStatesChan)

	go network.Module(
		localID,
		networkChns.LocalNodeStateChan,
		nodeStatesHandlerChns.RemoteNodeStatesChan,
		nodeStatesHandlerChns.NodeLostChan)

	go hallConsensus.ConsensusModule(
		localID,
		numFloors,
		hallConsensusChns.NewOrderChan,
		hallConsensusChns.ConfirmedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		)

	fmt.Println("(main) Started all modules");

	for {
		select {}
	}
}
