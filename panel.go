package auroraops

import "time"

type panelGroupState struct {
	status    string
	updatedAt time.Time
}

type panelGroup struct {
	thing        string
	panels       []int
	currentState panelGroupState
	action       Action
	onStart      string
	onStop       string
}

func (pg *panelGroup) start() error {
	return nil
}

func (pg *panelGroup) stop() error {
	return nil
}
