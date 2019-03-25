package datatypes

import (
	"../elevio"
)

type NodeID string

type Req struct {
	State ReqState
	AckBy []NodeID
}

type ReqState int

const (
	Unknown ReqState = iota
	Inactive
	PendingAck
	Confirmed
)

type HallOrdersMatrix [elevio.NumFloors][2]Req
type ConfirmedHallOrdersMatrix [elevio.NumFloors][2]bool

type CabOrdersList []Req
type CabOrdersMap map[NodeID]CabOrdersList
type ConfirmedCabOrdersList []bool
type ConfirmedCabOrdersMap map[NodeID]ConfirmedCabOrdersList

type AssignedOrdersMatrix [elevio.NumFloors][3]bool

// Todo change cabOrder as well
