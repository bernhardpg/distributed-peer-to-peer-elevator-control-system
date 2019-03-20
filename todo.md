### TODOS
- Change all chans into <- in parameters
- What to do with semicolon?
- Remember to fix issues with Peerlist and allNodeStates! Peers not on network needs to be removed by nodeStatesHandler!
- Make Golint happy
- Rename all channels structs as Channels
- Everything crashes when starting elevator outside boundaries
- Comment all code

### DONE
- Test spamming while initting
- fix problem where elev times out
- Change elevState to elevBehaviour or something
- Change name of statehandler to NodeStatesHandler or something
- Move state datatypes to FSM?
- Change channel names to be consistent
- Change name from optimalAssigner to optimalOrderAssigner
- Dont send from optass if equal result!

## NOTES
- Never have inf for in main, use for { select { }}
- ctrl + backslash opens stack trace --> genious
