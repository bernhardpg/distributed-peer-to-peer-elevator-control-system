package elevio

// LightsChannels ...
// Channels used for communication with the Elevator LightHandler
type LightsChannels struct {
	TurnOffLightsChan    chan ButtonEvent
	TurnOnLightsChan     chan ButtonEvent
	FloorIndicatorChan   chan int
	TurnOffHallLightChan chan ButtonEvent
	TurnOnHallLightChan  chan ButtonEvent
	TurnOffCabLightChan  chan ButtonEvent
	TurnOnCabLightChan   chan ButtonEvent
}

// LightHandler ...
// GoRoutine for controlling the lights of a single elevator
func LightHandler(
	numFloors int,
	TurnOffHallLight <-chan ButtonEvent,
	TurnOnHallLight <-chan ButtonEvent,
	TurnOffCabLight <-chan ButtonEvent,
	TurnOnCabLight <-chan ButtonEvent,
	FloorIndicator <-chan int) {
	// Turn off all lights at init
	for floor := 0; floor < numFloors; floor++ {
		for orderType := BT_HallUp; orderType <= BT_Cab; orderType++ {
			SetButtonLamp(orderType, floor, false)
		}
	}

	for {
		select {
		case a := <-TurnOffHallLight:
			SetButtonLamp(a.Button, a.Floor, false)
		case a := <-TurnOnHallLight:
			SetButtonLamp(a.Button, a.Floor, true)
		case a := <-TurnOffCabLight:
			SetButtonLamp(a.Button, a.Floor, false)
		case a := <-TurnOnCabLight:
			SetButtonLamp(a.Button, a.Floor, true)
		case a := <-FloorIndicator:
			SetFloorIndicator(a)
		}

	}
}
