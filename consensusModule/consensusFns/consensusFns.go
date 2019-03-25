package consensusFns

import (
	"../../datatypes"
)

func UniqueIDSlice(IDSlice []datatypes.NodeID) []datatypes.NodeID {

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

func containsElement(s []datatypes.NodeID, e datatypes.NodeID) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//Returns true if primaryList contains listFraction
func containsList(primaryList []datatypes.NodeID, listFraction []datatypes.NodeID) bool {
	for _, a := range listFraction {
		if !containsElement(primaryList, a) {
			return false
		}
	}
	return true
}

func Merge(pLocal *datatypes.Req, remote datatypes.Req, localID datatypes.NodeID, peersList []datatypes.NodeID) (bool, bool) {
	newConfirmedFlag := false
	newInactiveFlag := false

	switch (*pLocal).State {

	case datatypes.Inactive:
		if remote.State == datatypes.PendingAck {
			*pLocal = datatypes.Req{
				State: datatypes.PendingAck,
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
			}
		}

	case datatypes.PendingAck:
		(*pLocal).AckBy = UniqueIDSlice(append(remote.AckBy, localID))

		if (remote.State == datatypes.Confirmed) || containsList((*pLocal).AckBy, peersList) {
			(*pLocal).State = datatypes.Confirmed
			newConfirmedFlag = true
		}

	case datatypes.Confirmed:
		(*pLocal).AckBy = UniqueIDSlice(append(remote.AckBy, localID))

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
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
			}

		case datatypes.Confirmed:
			*pLocal = datatypes.Req{
				State: datatypes.Confirmed,
				AckBy: UniqueIDSlice(append(remote.AckBy, localID)),
				//Signaliser datatypes.Confirmed
			}
			newConfirmedFlag = true

		}
	}

	return newInactiveFlag, newConfirmedFlag
}
