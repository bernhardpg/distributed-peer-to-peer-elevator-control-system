/*
	TTK4145 Realtime Programming – Elevator Project 2019

	This project follows the Golint code style standards as currently
	used and developed by Google.
*/

package main

import (
	"./consensus"
	"./datatypes"
	"./elevio"
	"./fsm"
	"./network"
	"./nodestates"
	"./orderassignment"
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

	IDptr := flag.String("id", "1", "LocalID of the node")
	portPtr := flag.Int("port", 15657, "Port for connecting to elevator")

	flag.Parse()
	localID := "node_" + (datatypes.NodeID)(*IDptr)
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
		ArrivedAtFloorChan:          make(chan int),
		ToggleNetworkVisibilityChan: make(chan bool),
	}
	orderassignmentChns := orderassignment.Channels{
		LocallyAssignedOrdersChan: make(chan datatypes.AssignedOrdersMatrix, 2),
		PeerlistUpdateChan:        make(chan []datatypes.NodeID),
	}
	nodestatesChns := nodestates.Channels{
		LocalNodeStateChan: make(chan datatypes.NodeState, 2),
		AllNodeStatesChan:  make(chan datatypes.AllNodeStatesMap, 10),
		NodeLostChan:       make(chan datatypes.NodeID),
	}
	networkChns := network.Channels{
		LocalNodeStateChan:   make(chan datatypes.NodeState),
		RemoteNodeStatesChan: make(chan nodestates.NodeStateMsg, 2),
	}
	hallConsensusChns := consensus.HallOrderChannels{
		CompletedOrderChan:  make(chan int),
		NewOrderChan:        make(chan elevio.ButtonEvent),
		ConfirmedOrdersChan: make(chan datatypes.ConfirmedHallOrdersMatrix, 2),
		LocalOrdersChan:     make(chan datatypes.HallOrdersMatrix, 2),
		RemoteOrdersChan:    make(chan datatypes.HallOrdersMatrix, 10),
		PeerlistUpdateChan:  make(chan []datatypes.NodeID),
	}
	cabConsensusChns := consensus.CabOrderChannels{
		CompletedOrderChan:  make(chan int),
		NewOrderChan:        make(chan int),
		ConfirmedOrdersChan: make(chan datatypes.ConfirmedCabOrdersMap, 2),
		LocalOrdersChan:     make(chan datatypes.CabOrdersMap, 2),
		RemoteOrdersChan:    make(chan datatypes.CabOrdersMap, 10),
		PeerlistUpdateChan:  make(chan []datatypes.NodeID),
		LostPeerChan:        make(chan datatypes.NodeID),
	}
	// Note: Buffer are added to some of the channels to avoid issues with circular communication
	// and with many nodes transmitting on the network simultaneously.

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
		fsmChns.ToggleNetworkVisibilityChan,
		orderassignmentChns.LocallyAssignedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		cabConsensusChns.CompletedOrderChan,
		nodestatesChns.LocalNodeStateChan)

	go nodestates.Handler(
		localID,
		nodestatesChns.LocalNodeStateChan,
		nodestatesChns.AllNodeStatesChan,
		nodestatesChns.NodeLostChan,
		networkChns.LocalNodeStateChan,
		networkChns.RemoteNodeStatesChan)

	go orderassignment.OptimalAssigner(
		localID,
		numFloors,
		orderassignmentChns.PeerlistUpdateChan,
		orderassignmentChns.LocallyAssignedOrdersChan,
		hallConsensusChns.ConfirmedOrdersChan,
		cabConsensusChns.ConfirmedOrdersChan,
		nodestatesChns.AllNodeStatesChan)

	go network.Module(
		localID,
		fsmChns.ToggleNetworkVisibilityChan,
		networkChns.LocalNodeStateChan,
		networkChns.RemoteNodeStatesChan,
		nodestatesChns.NodeLostChan,
		orderassignmentChns.PeerlistUpdateChan,
		hallConsensusChns.LocalOrdersChan,
		hallConsensusChns.RemoteOrdersChan,
		hallConsensusChns.PeerlistUpdateChan,
		cabConsensusChns.LocalOrdersChan,
		cabConsensusChns.RemoteOrdersChan,
		cabConsensusChns.PeerlistUpdateChan,
		cabConsensusChns.LostPeerChan)

	go consensus.HallOrdersModule(
		localID,
		hallConsensusChns.NewOrderChan,
		hallConsensusChns.ConfirmedOrdersChan,
		hallConsensusChns.CompletedOrderChan,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		hallConsensusChns.LocalOrdersChan,
		hallConsensusChns.RemoteOrdersChan,
		hallConsensusChns.PeerlistUpdateChan)

	go consensus.CabOrdersModule(
		localID,
		cabConsensusChns.NewOrderChan,
		cabConsensusChns.ConfirmedOrdersChan,
		cabConsensusChns.CompletedOrderChan,
		iolightsChns.TurnOffCabLightChan,
		iolightsChns.TurnOnCabLightChan,
		cabConsensusChns.LocalOrdersChan,
		cabConsensusChns.RemoteOrdersChan,
		cabConsensusChns.PeerlistUpdateChan,
		cabConsensusChns.LostPeerChan)

	fmt.Println("(main) Started all goroutines.")

	for {
		select {}
	}
}
