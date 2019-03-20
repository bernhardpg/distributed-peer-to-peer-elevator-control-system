package main

import (
//	"fmt"
	"../../fsm"
	"fmt"
)
type Req struct {
	state ReqState;
	ackBy []fsm.NodeID
}

type ReqState int
const (
	Inactive ReqState = iota
	PendingAck
	Confirmed
	Unknown
)

func updateConfirmedHallOrders(locallyAssignedHallOrders [][] Req, confirmedHallOrders *[][] bool){
	for floor := range locallyAssignedHallOrders {
		for orderReq := range locallyAssignedHallOrders[floor] {
			if locallyAssignedHallOrders[floor][orderReq].state == Confirmed {
				
				(*confirmedHallOrders)[floor][orderReq] = true
			}else{
				(*confirmedHallOrders)[floor][orderReq] = false
			}
			
		}
	}

}



func main(){
	numFloors := 4
/*
	var localID = (fsm.NodeID)(2)
	inactiveReq := Req {
		state: Inactive, 
		ackBy: []fsm.NodeID{localID},
	}
	activeReq := Req {
		state: Confirmed, 
		ackBy: []fsm.NodeID{localID},
	}

*/
	var locallyAssignedHallOrders = make([][] Req, numFloors)
	var confirmedHallOrders = make([][] bool, numFloors)

	for floor := range locallyAssignedHallOrders {
		locallyAssignedHallOrders[floor] = make([] Req, 2)

		for orderReq := range locallyAssignedHallOrders[floor] {
			
			locallyAssignedHallOrders[floor][orderReq] = Req {
				state: Confirmed,
				ackBy: nil,
			}

			confirmedHallOrders[floor] = [] bool{false, false}
		}
	}
	
	fmt.Println(confirmedHallOrders)

	updateConfirmedHallOrders(locallyAssignedHallOrders, &confirmedHallOrders)
	fmt.Println(confirmedHallOrders)




}