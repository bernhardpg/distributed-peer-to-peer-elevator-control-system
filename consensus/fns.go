package consensus

import (
	"../datatypes"
)

// merge ...
// Forms the basis for all the consensus logic.
// Merges the wordview of a single local order request with a single remote order request.
// Possible order states:
//		Unknown - Nothing can be said with certainty about the order, will get overriden by all other states
//		Inactive - The order is completed and hence to be regarded as inactive
//		PendingAck - The order is pending acknowledgement from the other nodes on the network before it can be handled by a node
//		Confirmed - The order is confirmed by all nodes on the network and is ready to be served by a node
// @return newConfirmedFlag: the order was set to Confirmed
// @return newInactiveFlag: the order was set to Inactive
func merge(
	pLocal *datatypes.Req,
	remote datatypes.Req,
	localID datatypes.NodeID,
	peersList []datatypes.NodeID) (bool, bool) {

	newConfirmedFlag := false
	newInactiveFlag := false

	switch (*pLocal).State {

	case datatypes.Inactive:
		if remote.State == datatypes.PendingAck {
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
			}
		}

	case datatypes.PendingAck:
		(*pLocal).AckBy = uniqueIDSlice(append(remote.AckBy, localID))

		if (remote.State == datatypes.Confirmed) || containsList((*pLocal).AckBy, peersList) {
			(*pLocal).State = datatypes.Confirmed
			newConfirmedFlag = true
		}

	case datatypes.Confirmed:
		(*pLocal).AckBy = uniqueIDSlice(append(remote.AckBy, localID))

		if remote.State == datatypes.Inactive {
			*pLocal = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}
			newInactiveFlag = true
		}

	case datatypes.Unknown:
		switch remote.State {

		case datatypes.Inactive:
			*pLocal = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}
			newInactiveFlag = true

		case datatypes.PendingAck:
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
			}

		case datatypes.Confirmed:
			*pLocal = datatypes.Req{
				State: datatypes.Confirmed,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
				//Signaliser datatypes.Confirmed
			}
			newConfirmedFlag = true

		}
	}

	return newInactiveFlag, newConfirmedFlag
}

// TODO what does this do??
func uniqueIDSlice(IDSlice []datatypes.NodeID) []datatypes.NodeID {

	keys := make(map[datatypes.NodeID]bool)
	list := []datatypes.NodeID{}

	for _, entry := range IDSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// containtsID ...
// Returns whether or not the NodeID list passed as the first argument contains the NodeID passed as the second param
func containsID(s []datatypes.NodeID, e datatypes.NodeID) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// containsList ...
//Returns true if primaryList contains listFraction
func containsList(primaryList []datatypes.NodeID, listFraction []datatypes.NodeID) bool {
	for _, a := range listFraction {
		if !containsID(primaryList, a) {
			return false
		}
	}
	return true
}
