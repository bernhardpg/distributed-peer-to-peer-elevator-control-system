### TODOS
- Change all chans into <- in parameters
- What to do with semicolon?
- Make Golint happy
- Everything crashes when starting elevator outside boundaries
- Comment all code
- Change peerlist to set?
- Better names for LocalNodeStateChan ?
- Change all variable declarations from 'var' to i.e. 'localState := fsm.NodeState {}'
- Change NodeID to string?
- Rename neworder channels in hallConsensus and cabcons
	- Change both chans to elevio.ButtonEvent?
- IMPORTANT: Handle physical obstruction (timout in fsm, signal to network that node is to be regarded as offline) (Or, stop broadcasting for 20 secs, then reboot).
- BUGFIX: calculateNextOrder will loop forever if currOrder is removed by optimalOrderassigner (or what?)
- Blinking lights??

### DONE
- Deep copy?? Fixed with datatypes
- numFloors as param or global const?
- Rename generalConsensusModukle to consensusFunctions
- Change iolights -> lightsio
- numFloors as param or global const?
- Rename generalConsensusModukle to consensusFunctions
- Rename all channels structs as Channels
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
