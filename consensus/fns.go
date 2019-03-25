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
	peerlist []datatypes.NodeID) (bool, bool) {

	newConfirmedFlag := false
	newInactiveFlag := false

	// Set the new state of the local order based on the remote order
	switch (*pLocal).State {

	// Set the local order from Inactive to Pending and add the localID  if the remote order is Pending. 
	case datatypes.Inactive:
		if remote.State == datatypes.PendingAck {
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: uniqueIDSlice(append(remote.AckBy, localID)),
			}
		}

	// Set local order from Pending to Confirmed if all nodes have acknowledged the order or if the remote order is already Confirmed.
	// Add the localID to the ackBy list if the order is not yet Confirmed.
	case datatypes.PendingAck:

		if (remote.State == datatypes.Confirmed) || containsList((*pLocal).AckBy, peerlist) {
			(*pLocal).State = datatypes.Confirmed
			newConfirmedFlag = true
			break
		}
		(*pLocal).AckBy = uniqueIDSlice(append(remote.AckBy, localID))

	// Set the local order to Inactive if the remote order is Inactive
	case datatypes.Confirmed:
		if remote.State == datatypes.Inactive {
			*pLocal = datatypes.Req{
				State: datatypes.Inactive,
				AckBy: nil,
			}
			newInactiveFlag = true
		}

	// Blindly copy the remote order state (including ackBy list) if the local order is Unknown
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
			}
			newConfirmedFlag = true

		}
	}

	return newInactiveFlag, newConfirmedFlag
}

// uniqueIDSlice ...
// @return: A list of NodeID's not containing any duplicates.
// (Note that the returned list is not sorted, as this is not required by any other functionality).
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
// @return: Whether or not the NodeID list passed as the first argument contains the NodeID passed as the second param
func containsID(s []datatypes.NodeID, e datatypes.NodeID) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// containsList ...
// @return: true if primaryList contains listFraction, else otherwise
func containsList(primaryList []datatypes.NodeID, listFraction []datatypes.NodeID) bool {
	for _, a := range listFraction {
		if !containsID(primaryList, a) {
			return false
		}
	}
	return true
}
