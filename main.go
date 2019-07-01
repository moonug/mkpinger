package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli"
	routeros "gopkg.in/routeros.v2"
)

var (
	promgplost  *prometheus.GaugeVec
	promgmaxrtt *prometheus.GaugeVec
	promgminrtt *prometheus.GaugeVec
	promgavgrtt *prometheus.GaugeVec
	promst      *prometheus.GaugeVec
	cfg         *Config
)

func init() {
	promgplost = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mkmon",
			Subsystem: "ping",
			Name:      "loss",
			Help:      "Packet loss",
		},
		[]string{
			"name",
			"ip",
		},
	)
	promgmaxrtt = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mkmon",
			Subsystem: "ping",
			Name:      "max_rtt",
			Help:      "Max rtt",
		},
		[]string{
			"name",
			"ip",
		},
	)
	promgminrtt = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mkmon",
			Subsystem: "ping",
			Name:      "min_rtt",
			Help:      "Min rtt",
		},
		[]string{
			"name",
			"ip",
		},
	)
	promgavgrtt = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mkmon",
			Subsystem: "ping",
			Name:      "avg_rtt",
			Help:      "Avg rtt",
		},
		[]string{
			"name",
			"ip",
		},
	)
	promst = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "mkmon",
			Subsystem: "ping",
			Name:      "available",
			Help:      "Is host available",
		},
		[]string{
			"name",
			"ip",
			"status",
		},
	)
	prometheus.MustRegister(promgplost)
	prometheus.MustRegister(promgmaxrtt)
	prometheus.MustRegister(promgminrtt)
	prometheus.MustRegister(promgavgrtt)
	prometheus.MustRegister(promst)
}

func pinger(nw uint) {
	ch := make(chan Device)
	for i := 0; i < int(nw); i++ {
		go pingerWorker(ch)
	}
	for {
		for _, d := range cfg.Devices {
			if !d.Skip {
				ch <- d
			}
		}
	}
}

func pingerWorker(ch chan Device) {
	for d := range ch {

		mkp, err := routeros.Dial(cfg.Proxy.Address, cfg.Proxy.User, cfg.Proxy.Password)

		if err != nil {
			log.Println(err)
			continue
		}
		// log.Println("ping " + p.Host)
		r, err := mkp.Run("/ping", "=count=5", "=address="+d.Address)
		if err != nil {
			log.Println(err)
			continue
		}

		for _, ree := range r.Re {
			pl, err := strconv.ParseFloat(ree.Map["packet-loss"], 64)
			if err == nil {
				promgplost.WithLabelValues(d.Name, d.Address).Set(pl)
			}
			maxrtt, err := time.ParseDuration(ree.Map["max-rtt"])
			if err == nil {
				promgmaxrtt.WithLabelValues(d.Name, d.Address).Set(float64(maxrtt))
			}
			minrtt, err := time.ParseDuration(ree.Map["min-rtt"])
			if err == nil {
				promgminrtt.WithLabelValues(d.Name, d.Address).Set(float64(minrtt))
			}
			avgrtt, err := time.ParseDuration(ree.Map["avg-rtt"])
			if err == nil {
				promgavgrtt.WithLabelValues(d.Name, d.Address).Set(float64(avgrtt))
			}
			st := 0
			switch ree.Map["status"] {
			case "TTL exceeded":
				st = 2
			case "timeout":
				st = 2
			}
			promst.WithLabelValues(d.Name, d.Address, strconv.Itoa(st)).Set(float64(time.Now().Unix()))
		}
		mkp.Close()
	}
}

func exporter(c *cli.Context) error {
	cfgfile := c.String("config-file")
	if cfgfile == "" {
		return errors.New("Config file not set. Exiting..")
	}
	b, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return err
	}

	cfg, err = CfgLoad(bytes.NewReader(b))
	if err != nil {
		return err
	}
	go pinger(c.Uint("ping_workers"))
	return nil
}

func main() {
	app := cli.NewApp()

	app.Name = "mkpinger"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Flags = []cli.Flag{
		cli.UintFlag{
			Name:   "ping_workers,w",
			Value:  10,
			EnvVar: "PING_WORKERS",
		},
		cli.StringFlag{
			Name:   "metric_port",
			Value:  "9436",
			EnvVar: "METRIC_PORT",
		},
		cli.StringFlag{
			Name:   "config-file, f",
			EnvVar: "CONFIG_FILE",
		},
	}
	app.Action = exporter
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
