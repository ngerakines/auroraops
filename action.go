package auroraops

import (
	"context"
	"fmt"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/ngerakines/auroraops/client"
	log "github.com/sirupsen/logrus"
)

type Action interface {
	Start() error
	Stop(ctx context.Context) error
}

type noOpAction struct {
}

type solidFillAction struct {
	panels []int
	color  colorful.Color

	auroraClient client.AuroraClient
}

func NewNoOpAction() Action {
	return &noOpAction{}
}

func NewSolidFillAction(auroraClient client.AuroraClient, panels []int, color colorful.Color) (Action, error) {
	if !color.IsValid() {
		return nil, fmt.Errorf("error: invalid color")
	}
	return &solidFillAction{panels, color, auroraClient}, nil
}

func (a *solidFillAction) Start() error {
	r, g, b := a.color.Clamped().RGB255()
	for _, panel := range a.panels {
		a.auroraClient.SetPanelColor(byte(panel), byte(r), byte(g), byte(b))
	}
	return nil
}

func (a *solidFillAction) Stop(ctx context.Context) error {
	log.WithField("action", "solidfill").Info("Stopping")
	return nil
}

func (noOpAction) Start() error {
	return nil
}

func (noOpAction) Stop(ctx context.Context) error {
	log.WithField("action", "noOpAction").Info("Stopping")
	return nil
}

func ClearPanels(auroraClient client.AuroraClient, onstart string) error {
	color, err := colorful.Hex(onstart)
	if err != nil {
		return err
	}
	if !color.IsValid() {
		return fmt.Errorf("error: color %s is invalid", onstart)
	}
	r, g, b := color.Clamped().RGB255()
	panelInfo, err := auroraClient.GetInfo()
	if err != nil {
		return err
	}

	for _, panel := range panelInfo.Panels {
		log.WithFields(log.Fields{
			"panel": panel.ID,
			"hex":   color.Hex(),
			"r":     r,
			"g":     g,
			"b":     b,
		}).Debug("Setting panel colors")
		auroraClient.SetPanelColor(byte(panel.ID), byte(r), byte(g), byte(b))
	}

	return nil
}
