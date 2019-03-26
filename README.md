# TTK3145 Elevator Project Spring 2019

This project aims to set up a network of `n` cooperative elevators on a network, running on `m` floors. By default, it runs with three nodes and four floors.

### Network structure and information distribution
The nodes communicate on a peer-to-peer basis, without any master-slave configuration. Each node periodically broadcasts its own state information and all its information on all the orders currently in the system. Each node individually calculates which orders it should handle based on the information it receives from the other nodes on the network. A robust consensus logic is needed for this approach to work, ensuring that all the nodes arrives at the same conclusions at all times.

### Consensus logic
All the order requests will always have one of three states:
- 1. *Inactive*: The order is completed and hence to be regarded as Inactive.
- 2. *PendingAck*: The order is pending acknowledgement from the other nodes on the network before it can be served by a node.
- 3. *Confirmed*: The order is confirmed by all nodes on the network and is ready to be served by a node.
The consensus logic is based on the simple principle that an order can only advance to the next state and never backwards, ensuring that the wrong conclusion is never arrived at.

An additional state is added to allow overwriting uncertain information with information from the network:
- *Unknown*: Nothing can be said with certainty about the order, this state will get overriden by all other states.
This final state will allow the network work as a data backup for all the nodes.

### Program overview
Each node consists of the following modules:
- `(elevio) IOReader`:
    - Registers button presses and sensor data. The `IOReader` is responsible for informing the FSM with sensor data, and for informing the `OptimalAssigner` of new button presses.
- `(consensus) ConsensusModules` (`HallOrders` and `CabOrders`):
    - Will merge local knowledge about order statuses with remote knowledge supplied by the `NetworkModule`. Orders that are agreed upon by all peers will be sent to the `OptimalAssigner`.
- `(network) NetworkModule`:
    - Will broadcast all the local knowledge about all all the peers on the network, including states and orders. Responsible for informing the `ConsensusModules` of remote orders and the `NodeStatesHandler` of remote states. Keeps track of which peers are visible.
- `(orderassignment) OptimalAssigner`:
    - Receives all confirmed orders known to the node. Redirects local cab orders to the `FSM`, and filters through the hall orders this node should handle, based on information about all peers' states received by the `NodeStatesHandler`.
- `(nodestates) NodeStatesHandler`:
    - Redirects the local node state from the `FSM` to the `NetworkModule` and informs the `OptimalAssigner` about all nodes' states.
- `(fsm) Finite State Machine`:
    - Receives orders to handle from `OptimalAssigner` and informs the `ConsensusModules` when orders are completed.

Taking a look at the [datatypes](./datatypes/datatypes.go) is recommended to get an overview of the project before starting to look at the different modules.


### Disclaimer
The following code sections were entirely or partly copied from other works:
- The hall request assigner used by [OptimalAssigner function](./orderassignment/orderassignment.go) was made by github user [klasbo](https://github.com/klasbo) and handed out. The source and documentation can be found [here](https://github.com/TTK4145/Project-resources/tree/master/cost_fns/hall_request_assigner/).
- The [network driver](./network/driver) package is mostly identical to the one handed out, which can be found [here](https://github.com/TTK4145/Network-go/).
