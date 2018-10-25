/*
http://www.apache.org/licenses/LICENSE-2.0.txt


Copyright 2016 Intel Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package perfc

import (
	"strings"

	"github.com/alexbrainman/pc"

	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	log "github.com/sirupsen/logrus"
)

const (
	Name       = "perfc"
	Version    = 1
	PluginType = "publisher"
)

var memPCNames = []string{
	`\Memory\Available Mbytes`,
	`\Memory\Pages Input/sec`,
	`\Memory\Pages/sec`,
	`\Memory\Committed Bytes`,
	`\Memory\Commit Limit`,
	`\Memory\% Committed Bytes in Use`,
	`\Process(_Total)\Private Bytes`,
}

func init() {
	q, err := pc.OpenQuery("", 0)
	if err != nil {
		log.Error("OpenQuery failed: %v", err)
	}
	defer q.Close()

	cs := make(map[string]*pc.Counter)
	for _, name := range memPCNames {
		c, err := q.AddCounter(name, 0)
		if err != nil {
			log.Error("AddCounter(%s) failed: %v", name, err)
		}
		cs[name] = c
	}

	err = q.CollectData()
	if err != nil {
		log.Error("CollectData failed: %v", err)
	}

	//const format = PDH_FMT_DOUBLE
	//const format = PDH_FMT_LONG
	const format = pc.PDH_FMT_LARGE

	for name, c := range cs {
		log.Infof("Checking %v ...", name)

		ctype, rval, err := c.GetRawValue()
		if err != nil {
			log.Error("GetRawValue() failed: %v", err)
		}
		log.Infof("GetRawValue(): ctype=%v rval=%+v time=%v", ctype, rval, rval.Time())

		ctype, cval, err := c.GetFmtValue(format)
		switch err {
		case pc.PDH_INVALID_DATA:
			log.Infof("GetFmtValue(): specified counter instance does not exist, skipping")
		case nil:
			log.Infof("GetFmtValue(): ctype=%v cval=%+v", ctype, cval)
		default:
			log.Error("GetFmtValue() failed: %v", err)
		}
	}
}

// New returns an instance of the InfluxDB publisher
func New() *PerfcPublisher {
	return &PerfcPublisher{}
}

// PerfcPublisher the PAF snap publisher plugin
type PerfcPublisher struct {
}

type configuration struct {
	logLevel string
}

func getConfig(config plugin.Config) (configuration, error) {
	cfg := configuration{}
	var err error

	cfg.logLevel, err = config.GetString("log-level")
	if err != nil {
		cfg.logLevel = "undefined"
	}

	return cfg, nil
}

func (pp *PerfcPublisher) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	policy.AddNewStringRule([]string{""}, "log-level", false)
	return *policy, nil
}

// Publish publishes metric data to PAF database
func (pp *PerfcPublisher) Publish(metrics []plugin.Metric, pluginConfig plugin.Config) error {
	config, err := getConfig(pluginConfig)
	if err != nil {
		return err
	}

	logger := getLogger(config)

	for _, m := range metrics {
		logger.Infof("metric namespace %s", m.Namespace.String())
		if strings.HasSuffix(m.Namespace.String(), "/wait") {
			hash := m.Tags["sql"]
			log.Infof("hash %s", hash)
			// TODO: write perf counter
		}
	}

	return nil
}

func getLogger(config configuration) *log.Entry {
	logger := log.WithFields(log.Fields{
		"plugin-name":    Name,
		"plugin-version": Version,
		"plugin-type":    PluginType,
	})

	// default
	log.SetLevel(log.WarnLevel)

	levelValue := config.logLevel
	if levelValue != "undefined" {
		if level, err := log.ParseLevel(strings.ToLower(levelValue)); err == nil {
			log.SetLevel(level)
		} else {
			log.WithFields(log.Fields{
				"value":             strings.ToLower(levelValue),
				"acceptable values": "warn, error, debug, info",
			}).Warn("Invalid log-level config value")
		}
	}
	return logger
}
