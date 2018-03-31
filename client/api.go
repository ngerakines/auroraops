package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	baseURL    = "/api/beta"
	authURL    = "/new"
	infoURL    = "/%s"
	effectsURL = "/%s/effects"
)

type AuroraClient interface {
	Authorize() (string, error)
	GetInfo() (*HardwareInfo, error)
	SetPanelColor(panel, r, g, b byte) error
	Stop() error
}

type auroraClient struct {
	address string
	token   string

	ecLock *sync.Mutex
	ec     ExternalCommand
}

// New creates a new client.
func New(address string) (AuroraClient, error) {
	return NewWithToken(address, "")
}

// NewWithToken creates a new client with a provided token.
func NewWithToken(address, token string) (AuroraClient, error) {
	var ecLock sync.Mutex
	client := &auroraClient{
		address: address,
		token:   token,
		ecLock:  &ecLock,
	}
	if err := client.initExternalCommands(); err != nil {
		return nil, err
	}
	return client, nil
}

// Authorize will attempt to get an auth token from the device. Device must be put into pairing mode (hold down power button 5-7 seconds) for this to work.
// On success will return a valid auth token.
func (c *auroraClient) Authorize() (string, error) {
	dat := &struct {
		Token string `json:"auth_token"`
	}{}
	if err := c.request("POST", c.url(authURL), nil, dat); err != nil {
		return "", err
	}
	c.token = dat.Token
	return c.token, nil
}

// GetInfo returns hardware information about the nanoleaf aurora.
func (c *auroraClient) GetInfo() (*HardwareInfo, error) {
	dat := &HardwareInfo{}
	if err := c.request("GET", c.url(infoURL), nil, dat); err != nil {
		return nil, err
	}
	parts := strings.Split(dat.PanelLayout.Layout.LayoutData, " ")
	if len(parts) <= 2 {
		return dat, nil
	}
	n, _ := strconv.Atoi(parts[0])
	side, _ := strconv.Atoi(parts[1])
	if len(parts) < (n*4)+2 {
		return nil, fmt.Errorf("Invalid panel layout data")
	}
	for i := 0; i < n; i++ {
		p := &Panel{
			SideLength: side,
		}
		p.ID, _ = strconv.Atoi(parts[2+i*4])
		p.X, _ = strconv.Atoi(parts[3+i*4])
		p.Y, _ = strconv.Atoi(parts[4+i*4])
		p.Rotation, _ = strconv.Atoi(parts[5+i*4])
		dat.Panels = append(dat.Panels, p)
	}
	return dat, nil
}

func (c *auroraClient) initExternalCommands() error {
	c.ecLock.Lock()
	defer c.ecLock.Unlock()

	if c.ec != nil {
		return nil
	}

	r := strings.NewReader(`{ "write": {"command":"display","version":"1.0","animType":"extControl"} }`)
	dat := &struct {
		IP    string `json:"streamControlIpAddr"`
		Port  int    `json:"streamControlPort"`
		Proto string `json:"streamControlProtocol"`
	}{}
	if err := c.request("PUT", c.url(effectsURL), r, dat); err != nil {
		return errors.Wrap(err, "cannot initiate external command channel")
	}

	address := net.JoinHostPort(dat.IP, strconv.Itoa(dat.Port))
	c.ec = NewExternalCommander(address)
	return nil
}

func (c *auroraClient) SetPanelColor(panel, r, g, b byte) error {
	c.ecLock.Lock()
	defer c.ecLock.Unlock()

	if c.ec == nil {
		return fmt.Errorf("error: external command is not set")
	}

	return c.ec.Execute(panel, r, g, b)
}

func (c *auroraClient) Stop() error {
	c.ecLock.Lock()
	defer c.ecLock.Unlock()

	if c.ec == nil {
		return nil
	}

	return c.ec.Stop()
}

func (c *auroraClient) request(method, url string, reader io.Reader, target interface{}) error {
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		return errors.Wrapf(err, "could not create request: %s %s", method, url)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode == http.StatusForbidden {
		return fmt.Errorf("error: not properly authenticated to nanoleaf device")
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("error: bad status code from nanoleaf device: %d", response.StatusCode)
	}

	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(target)
}

func (c *auroraClient) url(path string) string {
	u := fmt.Sprintf("%s%s%s", c.address, baseURL, path)
	if strings.Contains(u, "%s") {
		u = fmt.Sprintf(u, c.token)
	}
	return u
}
