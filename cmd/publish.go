package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/openshift-metalkube/mdns-publisher/pkg/publisher"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	BindAddress string              `mapstructure:"bind_address"`
	Services    []publisher.Service `mapstructure:"service"`
}

var (
	cfgFile string
	conf    Config
	log     = logrus.New()
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publishes mDNS services",
	Run: func(cmd *cobra.Command, args []string) {
		ip := net.ParseIP(conf.BindAddress)
		log.WithFields(logrus.Fields{
			"ip": ip,
		}).Info("BindAddress")

		iface, err := publisher.FindIface(ip)
		log.WithFields(logrus.Fields{
			"name": iface.Name,
		}).Info("Binding interface")
		if err != nil {
			log.WithFields(logrus.Fields{
				"ip": ip,
			}).Fatal("Failed to find interface for specified ip")
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		waitGroup := &sync.WaitGroup{}
		shutdownChannel := make(chan struct{})
		for _, service := range conf.Services {
			waitGroup.Add(1)
			log.WithFields(logrus.Fields{
				"name":     service.Name,
				"hostname": service.HostName,
				"type":     service.SvcType,
				"domain":   service.Domain,
				"port":     service.Port,
				"ttl":      service.TTL,
			}).Info("Publishing service")
			go publisher.Publish(ip, iface, service, shutdownChannel, waitGroup)
		}

		<-sig
		close(shutdownChannel)
		log.Info("SIGTERM received, gracefully shutting down services...")
		waitGroup.Wait()
		log.Info("Done!")
	},
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		defaultCfgHome := path.Join("etc", "mdns")
		viper.AddConfigPath(defaultCfgHome)
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("[ERR] mdns-publish: Failed to read config %v", err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&conf); err != nil {
		log.Printf("[ERR] mdns-publish: Failed to parse config %v", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	publishCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/mdns/config.yaml)")
}

func Execute() {
	if err := publishCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
