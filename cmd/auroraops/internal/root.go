package internal

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/ngerakines/auroraops"
	"github.com/ngerakines/auroraops/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

var RootCmd = &cobra.Command{
	Use:   "auroraops",
	Short: "auroraops is a tool to relay information to your nanoleaf aurora.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mcpp",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auroraops v1.0.0 -- HEAD")
	},
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the update server",
	Run: func(cmd *cobra.Command, args []string) {

		thingManager := auroraops.NewThingManager()
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

		auroraClient, err := client.NewWithToken(viper.GetString("panel.url"), viper.GetString("panel.key"))
		if err != nil {
			log.WithError(err).Error("Could not create aurora client.")
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
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(serverCmd)

	// serverCmd.Flags().String("profile", "default", "The configuration profile to reference (default is 'default')")
	serverCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is ./auroraops.yaml)")

	viper.SetDefault("status.location", "http://localhost:8080/")
	viper.SetDefault("status.interval", 3)
	viper.SetDefault("validate.thing", true)
	viper.SetDefault("validate.status", true)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("AURORAOPS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// viper.BindPFlag("profile", serverCmd.Flags().Lookup("profile"))

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
