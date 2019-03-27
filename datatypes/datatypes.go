package datatypes

import (
	"../elevio"
)

// -------------
// Node Datatypes
// -------------

// NodeID ...
// Used to hold the ID of an elevator node all over the program.
type NodeID string

// NodeBehaviour ...
// Contains the current behaviour of a node.
// (Not to be confused with NodeState, which contains the
// NodeBehaviour as a property)
type NodeBehaviour int

// Possible node behaviours
const (
	// InitState ...
	// Used for initializing
	// (Either after a restart or after being physically obstructed)
	InitState NodeBehaviour = iota

	// IdleState ...
	// Node is standing still without orders.
	IdleState

	// DoorOpenState ...
	// Node is standing in a floor with the doors open.
	DoorOpenState

	// MovingState ...
	// Node is moving.
	MovingState
)

// NodeDir ...
// Which direction the node is currently moving.
// (Will also decide which direction the node will look for
// new orders first).
type NodeDir int

const (
	Up NodeDir = iota
	Down
)

// NodeState ...
// Contains all the state information of a node.
type NodeState struct {
	Behaviour NodeBehaviour
	Floor     int
	Dir       NodeDir
}

// AllNodeStatesMap ...
// Data structure used to contain the node states of all the
// nodes currently in the peerlist.
type AllNodeStatesMap map[NodeID]NodeState

// -------------
// Order Datatypes
// -------------

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
// Holds all the hall orders and their level of consensus on the network
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
// Holds all the cab orders (and their level of consensus) for all nodes currently in the system.
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
// Contains all the orders assigned to the current elevator node, both
// the hall orders and the cab orders, as boolean values.
type AssignedOrdersMatrix [elevio.NumFloors][3]bool
