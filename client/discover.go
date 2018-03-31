package client

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/mdns"
)

func Disocver(d time.Duration) ([]string, error) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	ch := make(chan *mdns.ServiceEntry, 5)
	defer close(ch)
	mdns.Lookup("_nanoleafapi._tcp", ch)

	timeout := time.After(d)
	possibilities := []string{}
	for {
		select {
		case entry := <-ch:
			possibility := "http://" + net.JoinHostPort(entry.AddrV4.String(), strconv.Itoa(entry.Port))
			possibilities = append(possibilities, possibility)
		case <-timeout:
			return possibilities, nil
		}
	}
}
