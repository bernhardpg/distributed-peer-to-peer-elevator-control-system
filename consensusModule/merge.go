package requestConsensus



func merge()
switch locallyAssignedHallOrders[floor][orderReq].state {

					case Inactive:
						if remoteHallOrders[floor][orderReq].state == PendingAck {
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: PendingAck, 
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
							}
						}

					case PendingAck:
						locallyAssignedHallOrders[floor][orderReq].ackBy = uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID))

						if remoteHallOrders[floor][orderReq].state == Confirmed ||  peersList == locallyAssignedHallOrders[floor][orderReq].ackBy {
							locallyAssignedHallOrders[floor][orderReq].state = Confirmed
							//Signaliser confirmed
						}
						

					case Confirmed:
						locallyAssignedHallOrders[floor][orderReq].ackBy = uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID))

						if remoteHallOrders[floor][orderReq].state == Inactive {
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Inactive,
								ackBy: nil,
							}
						}


					case Unknown:
						switch remoteHallOrders[floor][orderReq].state {


						case Inactive:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Inactive,
								ackBy: nil,
							}


						case PendingAck:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: PendingAck,
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
							}


						case Confirmed:
							locallyAssignedHallOrders[floor][orderReq] = Req {
								state: Confirmed,
								ackBy: uniqueIDSlice(append(remoteHallOrders[floor][orderReq].ackBy, localID)),
								//Signaliser confirmed
							}



						}

					}
				}

				
			}
		}