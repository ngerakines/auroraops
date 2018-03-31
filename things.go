package auroraops

import (
	"context"
	"fmt"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/ngerakines/auroraops/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	auroraClient client.AuroraClient
	Status       map[string]StatusConfigSet
	Things       map[string]ThingConfigSet
	panelGroups  map[string]*panelGroup
}

func NewThingManager(auroraClient client.AuroraClient) *ThingManager {
	return &ThingManager{
		auroraClient: auroraClient,
		Status:       make(map[string]StatusConfigSet),
		Things:       make(map[string]ThingConfigSet),
		panelGroups:  make(map[string]*panelGroup),
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

func (m *ThingManager) UpdateThing(thing, status string) error {
	pg, hasPanelGroup := m.panelGroups[thing]
	if !hasPanelGroup {
		return fmt.Errorf("error: no panel group for thing")
	}
	if pg.currentState.status == status {
		log.WithFields(log.Fields{
			"thing":  thing,
			"status": status,
		}).Info("Thing already has this status.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pg.action.Stop(ctx); err != nil {
		return err
	}

	newAction, err := m.actionForStatus(status, pg.panels, m.auroraClient)
	if err != nil {
		return err
	}
	pg.action = newAction
	err = pg.action.Start()
	if err != nil {
		return err
	}
	pg.currentState.status = status
	pg.currentState.updatedAt = time.Now()

	return nil
}

func (m *ThingManager) actionForStatus(status string, panels []int, ac client.AuroraClient) (Action, error) {
	statusConfig, hasStatus := m.Status[status]
	if !hasStatus {
		return nil, fmt.Errorf("no action for status: %s", status)
	}
	if statusConfig.Type == "solid" {
		color, err := colorful.Hex(statusConfig.Color)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid color: %s", statusConfig.Color)
		}
		return NewSolidFillAction(ac, panels, color)
	}
	return nil, fmt.Errorf("unsupported status type: %s", statusConfig.Type)
}
