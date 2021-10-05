/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package metrics ...
package metrics

import (
	"time"

	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
)

// FunctionLabel is a name of CSI plugin operation for which
// we measure duration
type FunctionLabel string

var pluginNamespace string

var (
	/**** Metrics related to controller ****/
	volumesCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: pluginNamespace,
			Name:      "volumes_count",
			Help:      "Total Number of volumes in the cluster.",
		},
	)

	volumesAttachedCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: pluginNamespace,
			Name:      "volumes_attached_count",
			Help:      "Total Number of volumes attached in the cluster ",
		},
	)
	functionDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pluginNamespace,
			Name:      "function_duration_seconds",
			Help:      "Time taken by various operation of Plugin",
		}, []string{"function"},
	)
	functionCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pluginNamespace,
			Name:      "functions_total",
			Help:      "The number of plugin operation  completeted successfully.",
		}, []string{"function"},
	)

	errorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pluginNamespace,
			Name:      "errors_total",
			Help:      "The number of plugin operation  failed due to an error.",
		}, []string{"type"},
	)
)

// RegisterAll registers all metrics.
func RegisterAll(namespace string) {
	pluginNamespace = namespace
	prometheus.MustRegister(volumesCount)
	prometheus.MustRegister(volumesAttachedCount)
	prometheus.MustRegister(functionDuration)
	prometheus.MustRegister(functionCount)
	prometheus.MustRegister(errorsCount)
}

// UpdateVolumeCount records number of volumes currently present in the cluster
func UpdateVolumeCount(podsCount int) {
	volumesCount.Set(float64(podsCount))
}

// UpdateVolumeAttachedCount records number of volumes currently attached in the cluster
func UpdateVolumeAttachedCount(podsCount int) {
	volumesAttachedCount.Set(float64(podsCount))
}

// UpdateDurationFromStart records the duration of the step identified by the
// label using start time
func UpdateDurationFromStart(logger *zap.Logger, label FunctionLabel, start time.Time) {
	duration := time.Since(start)
	logger.Info("Time to complete", zap.Float64(string(label), duration.Seconds()))
	UpdateDuration(label, duration)
}

// UpdateDuration records the duration of the step identified by the label
func UpdateDuration(label FunctionLabel, duration time.Duration) {
	functionDuration.WithLabelValues(string(label)).Set(duration.Seconds())
}

// RegisterError records any errors for any plugin operation.
func RegisterError(errType string, err error) {
	if err != nil {
		errType = err.Error() // TODO Get the error code
	}
	errorsCount.WithLabelValues(errType).Add(1.0)
}

// RegisterFunction records any errors for any plugin operation.
func RegisterFunction(label FunctionLabel) {
	functionCount.WithLabelValues(string(label)).Add(1.0)
}
