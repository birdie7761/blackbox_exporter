// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prober

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	yaml "gopkg.in/yaml.v2"

	"github.com/prometheus/blackbox_exporter/config"

	_ "github.com/mattn/go-oci8"
)

var dbs = map[string]*sql.DB{}
var l sync.Mutex

type DBConnConfig struct {
	Conns map[string]string `yaml:"conns"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline"`
}

var dbc *DBConnConfig = &DBConnConfig{}

func getSqlDB(module config.Module, logger log.Logger) (conn *sql.DB) {
	if _, ok := dbs[module.ORASQL.DNSName]; ok {
		return dbs[module.ORASQL.DNSName]
	}

	l.Lock()
	defer l.Unlock()
	if _, ok := dbs[module.ORASQL.DNSName]; ok {
		return dbs[module.ORASQL.DNSName]
	}

	dbc.loadDBConfig(module.ORASQL.DNSFile)
	// os.Setenv("NLS_LANG", "AMERICAN_AMERICA.ZHS16GBK")
	db, err := sql.Open("oci8", dbc.Conns[module.ORASQL.DNSName])
	db.SetMaxOpenConns(module.ORASQL.MaxOpenConns)
	if err != nil {
		level.Error(logger).Log("msg", "Error opening connection to database:", "err", err)
		return
	}
	db.SetConnMaxLifetime(time.Duration(1 * time.Hour))
	dbs[module.ORASQL.DNSName] = db
	return db
}

func ProbeORASQL(ctx context.Context, target string, module config.Module, registry *prometheus.Registry, logger log.Logger) (success bool) {

	var (
		durationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "probe_orasql_duration_seconds",
			Help: "Duration of process",
		})

		metricsGaugeVec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "probe_orasql_metrics",
			Help: "Duration of http request by phase, summed over all redirects",
		}, []string{"tags"})
	)

	defer func(begun time.Time) {
		durationGauge.Set(time.Since(begun).Seconds())
	}(time.Now())

	db := getSqlDB(module, logger)

	registry.MustRegister(durationGauge)
	registry.MustRegister(metricsGaugeVec)

	sql := "select " + target
	level.Info(logger).Log("msg", "sql", "sql", sql)
	rows, err := db.Query(sql)
	if err != nil {
		level.Error(logger).Log("msg", "Error Query:", "err", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var f1 string
		var f2 float64
		rows.Scan(&f1, &f2)
		metricsGaugeVec.WithLabelValues(f1).Add(f2)
		level.Info(logger).Log("tag", f1, "value", f2)
	}
	return true
}

func (sc *DBConnConfig) loadDBConfig(confFile string) (err error) {

	yamlFile, err := ioutil.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("Error reading config file: %s", err)
	}

	if err := yaml.Unmarshal(yamlFile, sc); err != nil {
		return fmt.Errorf("Error parsing config file: %s", err)
	}

	return nil
}

func (sc *DBConnConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain DBConnConfig
	if err := unmarshal((*plain)(sc)); err != nil {
		return err
	}

	return nil
}
