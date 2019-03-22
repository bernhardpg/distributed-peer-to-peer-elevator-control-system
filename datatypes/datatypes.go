package datatypes

import (
	"../elevio"
)

type NodeID int

type Req struct {
	State ReqState;
	AckBy []NodeID
}

type ReqState int
const (
	Unknown ReqState = iota
	Inactive
	PendingAck
	Confirmed
)

type HallOrdersMatrix [elevio.NumFloors][2] Req
type ConfirmedHallOrdersMatrix [elevio.NumFloors][2] bool

type AssignedOrdersMatrix [elevio.NumFloors][3] bool

// Todo change cabOrder as well

