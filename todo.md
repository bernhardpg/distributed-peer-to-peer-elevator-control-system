### TODOS
- Change all chans into <- in parameters
- What to do with semicolon?
- fix problem where elev times out
- Make Golint happy
- Rename all channels structs as Channels
- Everything crashes when starting elevator outside boundaries
- Comment all code
- Change peerlist to set?
- Change iolights -> lightsio
- Better names for LocalNodeStateChan ?
- Change all variable declarations from 'var' to i.e. 'localState := fsm.NodeState {}'
- Change NodeID to string?
- Multiple nodes: Cab and hall orders need to ble cleared the right way
- Rename neworder channels in hallConsensus and cabcons
	- Change both chans to elevio.ButtonEvent?
- Deep copy??
- numFloors as param or global const?

- Rename generalConsensusModukle to consensusFunctions
- IMPORTANT: Handle physical obstruction (timout in fsm, signal to network that node is to be regarded as offline) (Or, stop broadcasting for 20 secs, then reboot).
- BUGFIX: calculateNextOrder will loop forever if currOrder is removed by optimalOrderassigner (or what?)

### DONE
- Test spamming while initting
- Remember to fix issues with Peerlist and allNodeStates! Peers not on network needs to be removed by nodeStatesHandler!
- Change elevState to elevBehaviour or something
- Change name of statehandler to NodeStatesHandler or something
- Move state datatypes to FSM?
- Change channel names to be consistent
- Change name from optimalAssigner to optimalOrderAssigner
- Dont send from optass if equal result!

## NOTES
- Never have inf for in main, use for { select { }}
- ctrl + backslash opens stack trace --> genious
