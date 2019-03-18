package optimalAssigner

import (
	"os/exec"
	"../fsm"
	"fmt"
	"log"
	"os"
)


type OptimalAssignerChns struct {
	HallOrdersChan chan [][] bool
	CabOrdersChan chan [] bool
	ElevStateChan chan fsm.ElevStateObject
}
	/*

	Goal json object input:
	{
	    "hallRequests" : 
	        [[Boolean, Boolean], ...],
	    "states" : 
	        {
	            "id_1" : {
	                "behaviour"     : < "idle" | "moving" | "doorOpen" >
	                "floor"         : NonNegativeInteger
	                "direction"     : < "up" | "down" | "stop" >
	                "cabRequests"   : [Boolean, ...]
	            },
	            "id_2" : {...}
	        }
	}

*/

	/* Json output 

	{
    "id_1" : [[Boolean, Boolean], ...],
    "id_2" : ...
	}

	*/

/*func jsonPackager(hallRequests, elevStates) {

}*/

// TODO export comment
func Assigner(HallOrdersChan <-chan [][] bool, CabOrdersChan <-chan [] bool, ElevStateChan <-chan fsm.ElevStateObject) {
	for {
		select {
			case a := <- HallOrdersChan:
				fmt.Println(a);
			case a := <- CabOrdersChan:
				fmt.Println(a);
			case a := <- ElevStateChan:
				fmt.Println(a);
		}
	}
}

func runScript() {
	fmt.Println("Running optass");

	// Get current working directory
	dir, err := os.Getwd();

	if err != nil {
		log.Fatal(err);
	}

	// Run external script
	// TODO stop script from running in new window!
	cmd := exec.Command("gnome-terminal", "-e", dir + "/optimalAssigner/hall_request_assigner");
	out, err := cmd.Output();

	if err != nil {
		log.Fatal(err);
	}

	fmt.Println(out);
}