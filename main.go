/*
	TTK4145 Realtime Programming – Elevator Project 2019
	Developed by Sander Endresen, Arild Madshaven and Bernhard Paus Græsdal

	This project follows the Golint code style standards as currently used and developed by Google.
*/

package main

import (
	"./consensusModule/cabConsensus"
	"./consensusModule/hallConsensus"
	"./datatypes"
	"./elevio"
	"./fsm"
	"./network"
	"./nodeStatesHandler"
	"./optimalOrderAssigner"
	"flag"
	"fmt"
	"strconv"
)

func main() {

	// Set numFloors to equal the global const defined in elevio.go
	numFloors := elevio.NumFloors
	fmt.Println("(main) numFloors: ", numFloors)

	// ID and Port Handling
	// ------
	// Pass the ID in the command line with `go run main.go -id=our_id`
	// Pass the port number in the command line with `go run main.go -port=our_id`

	IDptr := flag.Int("id", 1, "LocalID of the node")
	portPtr := flag.Int("port", 15657, "Port for connecting to elevator")

	flag.Parse()
	localID := (datatypes.NodeID)(*IDptr)
	port := *portPtr

	fmt.Println("(main) localID:", localID)
	fmt.Println("(main) port:", port)

	// Connect to elevator through tcp (either hardware or simulator)
	// -----
	elevio.Init("localhost:" + strconv.Itoa(port))

	// Initialize channels
	// -----
	iolightsChns := elevio.LightsChannels{
		TurnOnLightsChan:     make(chan elevio.ButtonEvent),
		TurnOffLightsChan:    make(chan elevio.ButtonEvent),
		FloorIndicatorChan:   make(chan int),
		TurnOffHallLightChan: make(chan elevio.ButtonEvent),
		TurnOnHallLightChan:  make(chan elevio.ButtonEvent),
		TurnOffCabLightChan:  make(chan elevio.ButtonEvent),
		TurnOnCabLightChan:   make(chan elevio.ButtonEvent),
	}
	fsmChns := fsm.Channels{
		ArrivedAtFloorChan: make(chan int),
	}
	optimalOrderAssignerChns := optimalOrderAssigner.Channels{
		LocallyAssignedOrdersChan: make(chan datatypes.AssignedOrdersMatrix, 2),
		// Needs a buffer size bigger than one because the optimalOrderAssigner might send on this channel multiple times before FSM manages to receive!
	}
	nodeStatesHandlerChns := nodeStatesHandler.Channels{
		LocalNodeStateChan: make(chan fsm.NodeState),
		AllNodeStatesChan:  make(chan map[datatypes.NodeID]fsm.NodeState, 2),
		NodeLostChan:       make(chan datatypes.NodeID),
	}
	networkChns := network.Channels{
		LocalNodeStateChan:   make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan nodeStatesHandler.NodeStateMsg, 2),
	}
	hallConsensusChns := hallConsensus.Channels{
		CompletedOrderChan:  make(chan int),
		NewOrderChan:        make(chan elevio.ButtonEvent),
		ConfirmedOrdersChan: make(chan datatypes.ConfirmedHallOrdersMatrix),
		LocalOrdersChan:     make(chan datatypes.HallOrdersMatrix, 2),
		RemoteOrdersChan:    make(chan datatypes.HallOrdersMatrix),
		PeerlistUpdateChan:  make(chan []datatypes.NodeID),
	}
	cabConsensusChns := cabConsensus.Channels{
		CompletedOrderChan: make(chan int),
		NewOrderChan:       make(chan int),
	}

	// TODO Double check channel buffering!

	// Start modules
	// -----
	go elevio.IOReader(
		hallConsensusChns.NewOrderChan,
		cabConsensusChns.NewOrderChan,
		fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan)

	go elevio.LightHandler(
		numFloors,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		iolightsChns.TurnOffCabLightChan,
		iolightsChns.TurnOnCabLightChan,
		iolightsChns.FloorIndicatorChan)

	go fsm.StateMachine(
		numFloors,
		fsmChns.ArrivedAtFloorChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		cabConsensusChns.CompletedOrderChan,
		nodeStatesHandlerChns.LocalNodeStateChan)

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
		hallConsensusChns.RemoteOrdersChan,
		hallConsensusChns.PeerlistUpdateChan)

	go hallConsensus.ConsensusModule(
		localID,
		hallConsensusChns.NewOrderChan,
		hallConsensusChns.ConfirmedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		hallConsensusChns.LocalOrdersChan,
		hallConsensusChns.RemoteOrdersChan,
		hallConsensusChns.PeerlistUpdateChan)

	fmt.Println("(main) Started all modules")

	for {
		select {}
	}
}
