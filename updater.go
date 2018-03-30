package auroraops

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type updater struct {
	statusDestination chan StatusMap

	stop chan struct{}
}

func NewUpdater(stop chan struct{}, wg *sync.WaitGroup, statusDestination chan StatusMap) error {
	server := &updater{
		statusDestination,
		make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
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
			fields := log.Fields{}
			for k, v := range data {
				fields[k] = v
			}
			log.WithFields(fields).Info("received data")
		}
	}
}
