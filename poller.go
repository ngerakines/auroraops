package auroraops

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type poller struct {
	statusDestination chan StatusMap

	location string
	ticker   *time.Ticker
	stop     chan struct{}
}

type StatusMap map[string]string

func NewPoller(stop chan struct{}, wg *sync.WaitGroup, statusDestination chan StatusMap) error {
	location := viper.GetString("status.location")
	interval := viper.GetInt64("status.interval")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	server := &poller{
		statusDestination,
		location,
		ticker,
		make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		wg.Add(1)
		<-stop
		log.Info("Gracefully stopping poller.")
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Error("Error gracefully stopping poller.")
		} else {
			log.Info("Gracefully stopped poller.")
		}
		wg.Done()
	}()

	return server.Run()
}

func (p *poller) Shutdown(ctx context.Context) error {
	log.Info("poller stopping")
	close(p.stop)
	return nil
}

func (p *poller) Run() error {
	log.Info("poller starting")
	defer p.ticker.Stop()
	var running sync.WaitGroup
	running.Add(1)
	for {
		select {
		case <-p.stop:
			return nil
		case t := <-p.ticker.C:
			log.WithField("time", t).Debug("polling")
			var statusData map[string]string
			if err := p.poll(&statusData); err != nil {
				log.WithError(err).Error()
			} else {
				p.statusDestination <- statusData
			}
		}
	}
}

func (p *poller) poll(target interface{}) error {
	pollClient := &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := pollClient.Get(p.location)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(target)
}
