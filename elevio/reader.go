package elevio

import (
	"fmt"
)

// IOReader ...
// Main routine for reading io values and passing them on to corresponding channels
func IOReader(
	NewHallOrderChan chan<- ButtonEvent,
	NewCabOrderChan chan<- int,
	ArrivedAtFloorChan chan<- int,
	FloorIndicatorChan chan<- int) {

	drvButtons := make(chan ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	go pollButtons(drvButtons)
	go pollFloorSensor(drvFloors)
	go pollObstructionSwitch(drvObstr)
	go pollStopButton(drvStop)

	for {
		select {
		case a := <-drvButtons:

			if a.Button == BT_HallDown || a.Button == BT_HallUp {
				NewHallOrderChan <- a
			} else if a.Button == BT_Cab {
				//NewCabOrderChan <- a.Floor
			}

		case a := <-drvFloors:
			ArrivedAtFloorChan <- a
			FloorIndicatorChan <- a

		case a := <-drvObstr:
			fmt.Printf("(elevio) Obstruction: %+v\n", a)

		case a := <-drvStop:
			fmt.Printf("(elevio) Stop: %+v\n", a)
		}
	}
}
