package internal

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/ngerakines/auroraops"
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
			if err := auroraops.NewUpdater(stop, &wg, statusFerry); err != nil {
				log.WithError(err).Error("Error shutting down status updater.")
			} else {
				log.Info("status poller updater.")
			}
			wg.Done()
		}()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		<-c
		close(stop)
		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(serverCmd)

	// serverCmd.Flags().String("profile", "default", "The configuration profile to reference (default is 'default')")
	serverCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is ./auroraops.yaml)")

	viper.SetDefault("status.location", "http://localhost:8080/")
	viper.SetDefault("status.interval", 3)

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
