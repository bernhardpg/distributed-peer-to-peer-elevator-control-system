package datatypes

import (
	"../elevio"
)

// NodeID ...
// Used to hold the ID of an elevator node all over the program
type NodeID string

// Req ...
// Holds the level of consensus of a single order request on the network
// (both consensus state and all informed nodes)
type Req struct {
	State ReqState
	AckBy []NodeID
}

// ReqState ...
// Used to hold the different consensus states of a single order request
// on the network
type ReqState int

const (
	//	Unknown ...
	//	Nothing can be said with certainty about the order, will
	//	get overriden by all other states.
	Unknown ReqState = iota

	//	Inactive ...
	//	The order is completed and hence to be regarded as Inactive.
	Inactive

	//	PendingAck ...
	//	The order is pending acknowledgement from the other nodes
	//	on the network before it can be handled by a node.
	PendingAck

	//	Confirmed ...
	// 	The order is confirmed by all nodes on the network and is
	//	ready to be served by a node.
	Confirmed
)

// HallOrdersMatrix ...
// Used to represent all the hall orders and their state on the network
type HallOrdersMatrix [elevio.NumFloors][2]Req

// ConfirmedHallOrdersMatrix ...
// This is the datatype used for hall orders by the whole system except
// the consensus module.
// It will always contain all the Confirmed hall orders on the network as true,
// whereas all other orders on the network will be regarded as false.
type ConfirmedHallOrdersMatrix [elevio.NumFloors][2]bool

// CabOrdersList ...
// Holds all the cab orders and their level of consensus on the network
// for a single node.
type CabOrdersList []Req

// CabOrdersMap ...
// Holds all the cab orders for all nodes currently in the system.
type CabOrdersMap map[NodeID]CabOrdersList

// ConfirmedCabOrdersList ...
// Equivalent logic to that of ConfirmedHallOrdersMatrix.
type ConfirmedCabOrdersList []bool

// ConfirmedCabOrdersMap ...
// This is the datatype used for cab orders by the whole system except
// the consensus module.
// Contains one confirmed list for each node currently in the system.
type ConfirmedCabOrdersMap map[NodeID]ConfirmedCabOrdersList

// AssignedOrdersMatrix ...
// Contains all the orders assigned to the current elevator, both
// hall orders and cab orders, as boolean values.
type AssignedOrdersMatrix [elevio.NumFloors][3]bool
