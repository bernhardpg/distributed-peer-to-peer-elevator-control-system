package main

import (
	"./datatypes"
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
	
	numFloors := elevio.NumFloors;
	fmt.Println("(main) numFloors: ", numFloors)

	// ID Handling
	// ------

	// Pass the ID in the command line with `go run main.go -id=our_id`
	IDptr := flag.Int("id", 1, "LocalID of the node");
	// Pass the port number in the command line with `go run main.go -port=our_id`
	portPtr := flag.Int("port", 15657, "Port for connecting to elevator");

	flag.Parse()

	localID := (datatypes.NodeID)(*IDptr)
	port := *portPtr

	fmt.Println("(main) localID:", localID)
	fmt.Println("(main) port:", port)

	// Connect to elevator through tcp (either hardware or simulator)
	// -----
	elevio.Init("localhost:" + strconv.Itoa(port));


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
		LocallyAssignedOrdersChan: make(chan datatypes.AssignedOrdersMatrix, 2),
		// Needs a buffer size bigger than one because the optimalOrderAssigner might send on this channel multiple times before FSM manages to receive!
	}
	nodeStatesHandlerChns := nodeStatesHandler.NodeStatesHandlerChannels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		AllNodeStatesChan: make(chan map[datatypes.NodeID] fsm.NodeState, 2),
		NodeLostChan: make(chan datatypes.NodeID),
	}
	networkChns := network.Channels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan nodeStatesHandler.NodeStateMsg, 2),
	}
	hallConsensusChns := hallConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan elevio.ButtonEvent),
		ConfirmedOrdersChan: make(chan datatypes.ConfirmedHallOrdersMatrix),
		LocalOrdersChan: make(chan datatypes.HallOrdersMatrix, 2),
		RemoteOrdersChan: make(chan datatypes.HallOrdersMatrix),
	}
	cabConsensusChns := cabConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan int),
	}

	// TODO Double check channel buffering!


	// Start modules
	// -----
	go elevio.IOReader(
		hallConsensusChns.NewOrderChan,
		cabConsensusChns.NewOrderChan,
		fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan)

	go fsm.StateMachine(
		numFloors,
		fsmChns.ArrivedAtFloorChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		cabConsensusChns.CompletedOrderChan,
		nodeStatesHandlerChns.LocalNodeStateChan)

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
		nodeStatesHandlerChns.AllNodeStatesChan,
		nodeStatesHandlerChns.NodeLostChan,
		networkChns.LocalNodeStateChan,
		networkChns.RemoteNodeStatesChan)

	go optimalOrderAssigner.Assigner(
		localID,
		numFloors,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		hallConsensusChns.ConfirmedOrdersChan,
		nodeStatesHandlerChns.AllNodeStatesChan)

	go network.Module(
		localID,
		networkChns.LocalNodeStateChan,
		networkChns.RemoteNodeStatesChan,
		nodeStatesHandlerChns.NodeLostChan,
		hallConsensusChns.LocalOrdersChan,
		hallConsensusChns.RemoteOrdersChan)


	go hallConsensus.ConsensusModule(
		localID,
		hallConsensusChns.NewOrderChan,
		hallConsensusChns.ConfirmedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		hallConsensusChns.LocalOrdersChan,
		hallConsensusChns.RemoteOrdersChan,
		)

	fmt.Println("(main) Started all modules");

	for {
		select {}
	}
}
