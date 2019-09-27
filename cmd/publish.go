package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/openshift/mdns-publisher/pkg/publisher"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	BindAddress        string              `mapstructure:"bind_address"`
	CollisionAvoidance string              `mapstructure:"collision_avoidance"`
	Debug              bool                `mapstructure:"debug"`
	Services           []publisher.Service `mapstructure:"service"`
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
		if conf.Debug {
			log.SetLevel(logrus.DebugLevel)
			publisher.SetLogLevel(logrus.DebugLevel)
		}
		ip := net.ParseIP(conf.BindAddress)
		collisionStrategy, err := publisher.NewCollisionStrategy(conf.CollisionAvoidance)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error":           err,
				"accepted_values": strings.Join(publisher.CollisionStrategies(), " ")}).Fatal("Wrong collision_avoidance")
		}
		collisionStrategyName, err := collisionStrategy.String()
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed to get collision avoidance name")
		}
		log.WithFields(logrus.Fields{
			"ip":                  ip,
			"collision_avoidance": conf.CollisionAvoidance,
		}).Info("Publishing with settings")

		iface, err := publisher.FindIface(ip)
		if err != nil {
			log.WithFields(logrus.Fields{
				"ip": ip,
			}).Fatal("Failed to find interface for specified ip")
		}
		log.WithFields(logrus.Fields{
			"name": iface.Name,
		}).Info("Binding interface")

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		waitGroup := &sync.WaitGroup{}
		shutdownChannel := make(chan struct{})
		for _, service := range conf.Services {
			waitGroup.Add(1)
			err = service.AlterName(collisionStrategy)
			if err != nil {
				log.WithFields(logrus.Fields{
					"error":               err,
					"name":                service.Name,
					"collision_avoidance": collisionStrategyName,
				}).Fatal("Failed to apply service name collision avoidance")
			}
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
		ifaceChanged := make(chan struct{})
		go publisher.IfaceCheck(ip, iface, ifaceChanged)

		select {
			case <-sig:
			case <-ifaceChanged:
		}
		close(shutdownChannel)
		log.Info("Shutdown request received, gracefully shutting down services...")
		waitGroup.Wait()
		log.Info("Done!")
	},
}

func initConfig() {
	viper.SetDefault("collision_avoidance", "inaction")
	viper.SetDefault("debug", false)
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
	publishCmd.PersistentFlags().Bool("debug", true, "Set log level to Debug")
	viper.BindPFlag("debug", publishCmd.PersistentFlags().Lookup("debug"))
}

func Execute() {
	if err := publishCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
