package client

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	tomb "gopkg.in/tomb.v2"
)

type PanelColorCommand struct {
	ID, R, G, B byte
}

type externalCommand struct {
	address string
	ch      chan *PanelColorCommand
	t       tomb.Tomb
	mu      sync.Mutex
}

type ExternalCommand interface {
	Execute(panel, r, g, b byte) error
	Stop() error
}

func NewExternalCommander(address string) ExternalCommand {
	ec := &externalCommand{
		address: address,
		ch:      make(chan *PanelColorCommand),
	}
	ec.t.Go(ec.loop)
	return ec
}

func (ec *externalCommand) Execute(panel, r, g, b byte) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	if ec.t.Alive() {
		ec.ch <- &PanelColorCommand{panel, r, g, b}
	}
	return nil
}

func (ec *externalCommand) loop() error {
	updates := map[byte][]byte{}
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case t := <-ticker.C:
			log.WithFields(log.Fields{
				"time":  t,
				"count": len(updates),
			}).Debug("publishing commands")
			if len(updates) > 0 {
				conn, err := net.Dial("udp", ec.address)
				if err != nil {
					log.WithError(err).Error("unable to connect to aurora")
					continue
				}
				conn.Write(buildUDPPacket(updates))
				conn.Close()
				updates = map[byte][]byte{}
			}
		case command := <-ec.ch:
			updates[command.ID] = []byte{1, command.R, command.G, command.B, 0, 1}
		case <-ec.t.Dying():
			log.WithField("count", len(updates)).Info("Stopping aurora client")
			close(ec.ch)
			return nil
		}
	}
}

func (ec *externalCommand) Stop() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.t.Kill(nil)
	return ec.t.Wait()
}

func externalControl(address string, ch <-chan *PanelColorCommand, stop chan struct{}) {
	updates := map[byte][]byte{}
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-stop:
			log.WithField("length", len(updates)).Info()
			return
		case t := <-ticker.C:
			log.WithField("time", t).Debug("polling")
			if len(updates) > 0 {
				conn, err := net.Dial("udp", address)
				if err != nil {
					log.WithError(err).Error("unable to connect to aurora")
					continue
				}
				conn.Write(buildUDPPacket(updates))
				conn.Close()
				updates = map[byte][]byte{}
			}
		case command := <-ch:
			updates[command.ID] = []byte{1, command.R, command.G, command.B, 0, 1}
		}
	}
}

func buildUDPPacket(dat map[byte][]byte) []byte {
	buf := make([]byte, 0, (len(dat)*6)+1)
	buf = append(buf, byte(len(dat)))
	for id, pt := range dat {
		buf = append(buf, id)
		buf = append(buf, pt...)
	}
	return buf
}
