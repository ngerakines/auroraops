package internal

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"time"

	colorful "github.com/lucasb-eyer/go-colorful"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/ngerakines/auroraops"
	"github.com/ngerakines/auroraops/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

var (
	cfgFile string
)

var RootCmd = &cobra.Command{
	Use:   "auroraops",
	Short: "auroraops is a tool to relay information to your nanoleaf aurora.",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Performs first-time setup.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hosts, err := client.Disocver(5 * time.Second)
		if err != nil {
			log.WithError(err).Error("Could not discover aurora.")
			os.Exit(1)
		}
		if len(hosts) == 0 {
			log.Error("No aurora hosts found.")
			os.Exit(1)
		}
		if len(hosts) > 1 {
			log.WithField("hosts", hosts).Error("More than one aurora host found.")
			os.Exit(1)
		}

		client, err := client.New(hosts[0])
		if err != nil {
			log.WithError(err).Error("Could not discover aurora.")
			os.Exit(1)
		}

		fmt.Println("Need to authenticate to device. Hold power button to enter pairing mode. Press enter when ready.")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		token, err := client.Authorize()
		if err != nil {
			log.WithError(err).Error("Unable to authenticate")
			os.Exit(1)
		}

		type Config struct {
			Name string `json:"name" yaml:"name"` // Supporting both JSON and YAML.
			Age  int    `json:"name" yaml:"name"`
		}

		data := struct {
			Panel struct {
				URL string `yaml:"url"`
				Key string `yaml:"key"`
			} `yaml:"panel"`
		}{
			Panel: struct {
				URL string `yaml:"url"`
				Key string `yaml:"key"`
			}{
				URL: hosts[0],
				Key: token,
			},
		}

		d, err := yaml.Marshal(&data)
		if err != nil {
			log.WithFields(log.Fields{
				"token": token,
				"host":  hosts[0],
			}).WithError(err).Error("Unable to compose yaml")
			os.Exit(1)
		}

		if err := ioutil.WriteFile(args[0], d, 0644); err != nil {
			log.WithFields(log.Fields{
				"token": token,
				"host":  hosts[0],
			}).WithError(err).Error("Unable to compose yaml")
			os.Exit(1)
		}
		log.WithFields(log.Fields{
			"file":  args[0],
			"token": token,
			"host":  hosts[0],
		}).Info("Configuration file initialized")
	},
}

var colors = map[string]string{
	"white":   "#ffffff",
	"grey":    "#808080",
	"red":     "#ff0000",
	"maroon":  "#80000",
	"yellow":  "#ffff00",
	"lime":    "#00ff00",
	"green":   "#00ffff",
	"aqua":    "#00FFFF",
	"teal":    "#008080",
	"blue":    "#0000ff",
	"fuchsia": "#ff00ff",
	"purple":  "#800080",
	"olive":   "#808000",
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Run the server.",
	Run: func(cmd *cobra.Command, args []string) {
		auroraClient, err := client.NewWithToken(viper.GetString("panel.url"), viper.GetString("panel.key"))
		if err != nil {
			log.WithError(err).Error("Could not create aurora client.")
			os.Exit(1)
		}
		panelInfo, err := auroraClient.GetInfo()
		if err != nil {
			log.WithError(err).Error("Could not get panel info")
			os.Exit(1)
		}

		colorNames := make([]string, 0, len(colors))
		colorfulColors := make([]colorful.Color, 0, len(colors))
		for name, hex := range colors {
			colorNames = append(colorNames, name)
			color, err := colorful.Hex(hex)
			if err != nil {
				log.WithError(err).Errorf("Could not process color %s %s", name, hex)
				os.Exit(1)
			}
			colorfulColors = append(colorfulColors, color)
		}

		colorIndex := 0
		for _, panel := range panelInfo.Panels {
			if colorIndex > len(colorNames) {
				colorIndex = 0
			}
			color := colorfulColors[colorIndex]
			colorName := colorNames[colorIndex]
			fmt.Printf("Setting panel %d to %s\n", panel.ID, colorName)
			r, g, b := color.Clamped().RGB255()
			auroraClient.SetPanelColor(byte(panel.ID), byte(r), byte(g), byte(b))

			colorIndex = colorIndex + 1
		}

		time.Sleep(2 * time.Second)

		if err = auroraClient.Stop(); err != nil {
			log.WithError(err).Error("Could not stop aurora client.")
		}
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the server.",
	Run: func(cmd *cobra.Command, args []string) {

		auroraClient, err := client.NewWithToken(viper.GetString("panel.url"), viper.GetString("panel.key"))
		if err != nil {
			log.WithError(err).Error("Could not create aurora client.")
			os.Exit(1)
		}

		thingManager := auroraops.NewThingManager(auroraClient)
		if err := viper.UnmarshalKey("status", &thingManager.Status); err != nil {
			log.WithError(err).Error("Could not parse status configuration.")
			os.Exit(1)
		}
		if err := viper.UnmarshalKey("things", &thingManager.Things); err != nil {
			log.WithError(err).Error("Could not parse status configuration.")
			os.Exit(1)
		}

		if err := thingManager.Init(); err != nil {
			log.WithError(err).Error("Invalid thing configuration.")
			os.Exit(1)
		}

		onstart := viper.GetString("onstart")
		log.WithField("color", onstart).Info("Clearing panels")
		if onstart != "" {
			if err := auroraops.ClearPanels(auroraClient, onstart); err != nil {
				log.WithError(err).Error("Could not clear panels.")
				os.Exit(1)
			}
			log.Info("Panels cleared.")
		}

		// pretty.Println(thingManager)

		if err = thingManager.StartAll(auroraClient); err != nil {
			log.WithError(err).Error("Could not start things.")
			os.Exit(1)
		}

		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)

		statusFerry := make(chan auroraops.StatusMap)

		go func() {
			if err := auroraops.NewPoller(stop, &wg, statusFerry); err != nil {
				log.WithError(err).Error("Error shutting down status poller.")
			} else {
				log.Info("status poller stopped.")
			}
			wg.Done()
		}()

		go func() {
			if err := auroraops.NewUpdater(stop, &wg, statusFerry, thingManager, auroraClient); err != nil {
				log.WithError(err).Error("Error shutting down status updater.")
			} else {
				log.Info("status updater stopped.")
			}
			wg.Done()
		}()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		close(stop)
		wg.Wait()

		if err = thingManager.StopAll(auroraClient); err != nil {
			log.WithError(err).Error("Could not stop things.")
		}

		time.Sleep(2 * time.Second)

		if err = auroraClient.Stop(); err != nil {
			log.WithError(err).Error("Could not stop aurora client.")
		}

	},
}

func init() {
	log.SetLevel(log.InfoLevel)
	RootCmd.AddCommand(initCmd)
	RootCmd.AddCommand(infoCmd)
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is ./auroraops.yaml)")

	viper.SetDefault("status.location", "http://localhost:8080/")
	viper.SetDefault("status.interval", 3)
	viper.SetDefault("validate.thing", true)
	viper.SetDefault("validate.status", true)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("AURORAOPS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(".")
		viper.AddConfigPath(path.Join(home, ".auroraops"))
		viper.AddConfigPath("/etc/auroraops/")
		viper.SetConfigName("auroraops")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
	}
}
