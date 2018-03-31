package auroraops

import (
	"context"
	"sync"
	"time"

	"github.com/ngerakines/auroraops/client"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

type updater struct {
	statusDestination chan StatusMap
	thingManager      *ThingManager
	auroraClient      client.AuroraClient

	stop chan struct{}
}

type thingStatusPair struct {
	thing  string
	status string
}

func NewUpdater(stop chan struct{}, wg *sync.WaitGroup, statusDestination chan StatusMap, thingManager *ThingManager, auroraClient client.AuroraClient) error {
	server := &updater{
		statusDestination,
		thingManager,
		auroraClient,
		make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		wg.Add(1)
		<-stop
		log.Info("Gracefully stopping updater.")
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Error("Error gracefully stopping updater.")
		} else {
			log.Info("Gracefully stopped updater.")
		}
		wg.Done()
	}()

	return server.Run()
}

func (p *updater) Shutdown(ctx context.Context) error {
	log.Info("updater stopping")
	close(p.stop)
	return nil
}

func (p *updater) Run() error {
	log.Info("updater starting")

	var running sync.WaitGroup
	running.Add(1)
	for {
		select {
		case <-p.stop:
			return nil
		case data := <-p.statusDestination:
			thingPairs, err := p.validate(data)
			if err != nil {
				log.WithError(err).Warn()
			}
			fields := log.Fields{}
			for _, thingPair := range thingPairs {
				fields[thingPair.thing] = thingPair.status
			}
			log.WithFields(fields).Info("received data")
			for _, thingPair := range thingPairs {
				if err := p.thingManager.UpdateThing(thingPair.thing, thingPair.status); err != nil {
					log.WithError(err).WithFields(log.Fields{
						"thing":  thingPair.thing,
						"status": thingPair.status,
					}).Error("Could not update thing.")
				}
			}
		}
	}
}

func (p *updater) validate(statusData StatusMap) ([]thingStatusPair, error) {
	warnOnUnknownStatus := viper.GetBool("validate.status")
	warnOnUnknownThing := viper.GetBool("validate.thing")

	pairs := []thingStatusPair{}
	things := []string{}
	statuses := []string{}
	for thing, status := range statusData {
		if warnOnUnknownThing && !containsString(things, thing) {
			things = append(things, thing)
		}
		if warnOnUnknownStatus && !containsString(statuses, status) {
			statuses = append(statuses, status)
		}
	}
	if warnOnUnknownThing {
		for _, thing := range things {
			if _, ok := p.thingManager.Things[thing]; !ok {
				log.WithField("thing", thing).Warn("Unexexpected thing found.")
			}
		}
	}
	if warnOnUnknownStatus {
		for _, status := range statuses {
			if _, ok := p.thingManager.Status[status]; !ok {
				log.WithField("status", status).Warn("Unexexpected status found.")
			}
		}
	}
	for thing, status := range statusData {
		_, thingOK := p.thingManager.Things[thing]
		_, statusOK := p.thingManager.Status[status]
		if thingOK && statusOK {
			pairs = append(pairs, thingStatusPair{thing, status})
		}
	}

	return pairs, nil
}

func containsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func containsInt(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
