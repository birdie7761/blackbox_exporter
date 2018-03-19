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
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/blackbox_exporter/config"

	_ "github.com/mattn/go-oci8"
)

var db *sql.DB
var isInitDB bool
var l sync.Mutex

func initSqlDB(module config.Module, logger log.Logger) {
	if !isInitDB {
		l.Lock()
		defer l.Unlock()
		if isInitDB {
			return
		}
		var err error
		// os.Setenv("NLS_LANG", "AMERICAN_AMERICA.ZHS16GBK")
		db, err = sql.Open("oci8", module.ORASQL.DNS)
		db.SetMaxOpenConns(module.ORASQL.MaxOpenConns)
		if err != nil {
			level.Error(logger).Log("msg", "Error opening connection to database:", "err", err)
			return
		}
		db.SetConnMaxLifetime(time.Duration(1 * time.Hour))
		isInitDB = true
	}
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

	initSqlDB(module, logger)

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
		level.Info(logger).Log("msg", "key", f1, "value", f2)
	}
	return true
}
