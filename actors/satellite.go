package actor

import "main"

b := main.NUMBER_OF_ORBITS

def Satellite() {
	for i := 0; i < main.NUMBER_OF_SATELLITES; i++ {
		for j := 0; j < b; j++ {
			Satellite()
		}
	}
}