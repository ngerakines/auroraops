package client

type HardwareInfo struct {
	Name            string `json:"name"`
	SerialNo        string `json:"serialNo"`
	Manufacturer    string `json:"manufacturer"`
	FirmwareVersion string `json:"firmwareVersion"`
	Model           string `json:"model"`
	State           struct {
		On struct {
			Value bool `json:"value"`
		} `json:"on"`
		Brightness struct {
			Value int `json:"value"`
			Max   int `json:"max"`
			Min   int `json:"min"`
		} `json:"brightness"`
		Hue struct {
			Value int `json:"value"`
			Max   int `json:"max"`
			Min   int `json:"min"`
		} `json:"hue"`
		Sat struct {
			Value int `json:"value"`
			Max   int `json:"max"`
			Min   int `json:"min"`
		} `json:"sat"`
		Ct struct {
			Value int `json:"value"`
			Max   int `json:"max"`
			Min   int `json:"min"`
		} `json:"ct"`
		ColorMode string `json:"colorMode"`
	} `json:"state"`
	Effects struct {
		Select string   `json:"select"`
		List   []string `json:"list"`
	} `json:"effects"`
	Panels      []*Panel `json:"panels"`
	PanelLayout struct {
		Layout struct {
			LayoutData string `json:"layoutData"`
		} `json:"layout"`
		GlobalOrientation struct {
			Value int `json:"value"`
			Max   int `json:"max"`
			Min   int `json:"min"`
		} `json:"globalOrientation"`
	} `json:"panelLayout"`
}

type Panel struct {
	ID         int `json:"id"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Rotation   int `json:"rotation"`
	SideLength int `json:"length"`
}
