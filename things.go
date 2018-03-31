package auroraops

import (
	"context"
	"fmt"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/ngerakines/auroraops/client"
)

type StatusConfigSet struct {
	Color string `mapstructure:"color"`
	Type  string `mapstructure:"type"`
}

type ThingConfigSet struct {
	Panels  []int  `mapstructure:"panels"`
	OnStart string `mapstructure:"onstart"`
	OnStop  string `mapstructure:"onstop"`
}

type ThingManager struct {
	Status      map[string]StatusConfigSet
	Things      map[string]ThingConfigSet
	panelGroups map[string]*panelGroup
}

func NewThingManager() *ThingManager {
	return &ThingManager{
		Status:      make(map[string]StatusConfigSet),
		Things:      make(map[string]ThingConfigSet),
		panelGroups: make(map[string]*panelGroup),
	}
}

func (m *ThingManager) validate() error {
	panels := []int{}
	for _, thingConfig := range m.Things {
		for _, panel := range thingConfig.Panels {
			if containsInt(panels, panel) {
				return fmt.Errorf("Pannel %d is referenced in multiple things.", panel)
			}
			panels = append(panels, panel)
		}
	}
	return nil
}

func (m *ThingManager) Init() error {
	if err := m.validate(); err != nil {
		return err
	}
	for thing, thingInfo := range m.Things {
		m.panelGroups[thing] = &panelGroup{
			thing:  thing,
			panels: thingInfo.Panels,
			currentState: panelGroupState{
				status:    "",
				updatedAt: time.Now(),
			},
			action:  NewNoOpAction(),
			onStart: thingInfo.OnStart,
			onStop:  thingInfo.OnStop,
		}
	}
	return nil
}

func (m *ThingManager) StartAll(auroraClient client.AuroraClient) error {
	for _, panelGroup := range m.panelGroups {
		if panelGroup.onStart != "" {
			color, err := colorful.Hex(panelGroup.onStart)
			if err != nil {
				return err
			}
			action, err := NewSolidFillAction(auroraClient, panelGroup.panels, color)
			if err != nil {
				return err
			}
			panelGroup.action = action
			err = panelGroup.action.Start()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *ThingManager) StopAll(auroraClient client.AuroraClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, panelGroup := range m.panelGroups {
		if err := panelGroup.action.Stop(ctx); err != nil {
			return err
		}
		if panelGroup.onStop != "" {
			color, err := colorful.Hex(panelGroup.onStop)
			if err != nil {
				return err
			}
			action, err := NewSolidFillAction(auroraClient, panelGroup.panels, color)
			if err != nil {
				return err
			}
			panelGroup.action = action
			err = panelGroup.action.Start()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
