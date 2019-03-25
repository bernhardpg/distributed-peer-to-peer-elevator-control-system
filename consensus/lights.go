package consensus

// ----------
// Setting the elevator lights
// ----------

import (
	"../elevio"
)

// TODO generalize light functions

func setHallLight(currFloor int, orderType int, TurnOnHallLightChan chan<- elevio.ButtonEvent) {
	buttonDir := (elevio.ButtonType)(orderType)

	buttonToIlluminate := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: buttonDir,
	}
	TurnOnHallLightChan <- buttonToIlluminate
}

func clearHallLights(currFloor int, TurnOffHallLightChan chan<- elevio.ButtonEvent) {

	callUpAtFloor := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_HallUp,
	}

	callDownAtFloor := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_HallDown,
	}

	TurnOffHallLightChan <- callDownAtFloor
	TurnOffHallLightChan <- callUpAtFloor
}

func setCabLight(currFloor int, TurnOnCabLightChan chan<- elevio.ButtonEvent) {

	buttonToIlluminate := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOnCabLightChan <- buttonToIlluminate
}

func clearCabLight(currFloor int, TurnOffCabLightChan chan<- elevio.ButtonEvent) {

	buttonToClear := elevio.ButtonEvent{
		Floor:  currFloor,
		Button: elevio.BT_Cab,
	}

	TurnOffCabLightChan <- buttonToClear
}
