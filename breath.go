package auroraops

import (
	"context"
	"sync"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/ngerakines/auroraops/client"
	log "github.com/sirupsen/logrus"
	tomb "gopkg.in/tomb.v2"
)

type breathAction struct {
	panels   []int
	colors   []colorful.Color
	position int

	auroraClient client.AuroraClient

	t  tomb.Tomb
	mu sync.Mutex
}

type gradientTable []struct {
	Col colorful.Color
	Pos float64
}

func NewBreathAction(auroraClient client.AuroraClient, panels []int, to, from colorful.Color, seconds int) (Action, error) {
	log.Info("New breath action")
	steps := seconds * 20
	keypoints := gradientTable{
		{from, 0.0},
		{to, 0.2},
		{to, 0.8},
		{from, 1.0},
	}
	colors := []colorful.Color{}
	for y := steps; y >= 0; y-- {
		c := keypoints.getInterpolatedColorFor(float64(y) / float64(steps))
		colors = append(colors, c)
	}
	ba := &breathAction{
		panels:       panels,
		colors:       colors,
		position:     0,
		auroraClient: auroraClient,
	}
	return ba, nil
}

func (a *breathAction) loop() error {
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case t := <-ticker.C:
			if a.position >= len(a.colors) {
				a.position = 0
			}
			color := a.colors[a.position]
			log.WithFields(log.Fields{
				"t":        t,
				"action":   "breath",
				"position": a.position,
				"color":    color.Hex(),
			}).Debug("Tick")
			r, g, b := color.Clamped().RGB255()
			for _, panel := range a.panels {
				a.auroraClient.SetPanelColor(byte(panel), byte(r), byte(g), byte(b))
			}
			a.position = a.position + 1
		case <-a.t.Dying():
			return nil
		}
	}
}

func (ba *breathAction) Start() error {
	ba.t.Go(ba.loop)
	return nil
}

func (ba *breathAction) Stop(ctx context.Context) error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	log.WithField("action", "breath").Info("Stopping")
	ba.t.Kill(nil)
	return ba.t.Wait()
}

func (gt gradientTable) getInterpolatedColorFor(t float64) colorful.Color {
	for i := 0; i < len(gt)-1; i++ {
		c1 := gt[i]
		c2 := gt[i+1]
		if c1.Pos <= t && t <= c2.Pos {
			// We are in between c1 and c2. Go blend them!
			t := (t - c1.Pos) / (c2.Pos - c1.Pos)
			return c1.Col.BlendHcl(c2.Col, t).Clamped()
		}
	}
	return gt[len(gt)-1].Col
}
