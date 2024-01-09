package exporter

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shiningrush/fastflow/pkg/entity"
	"github.com/shiningrush/fastflow/pkg/event"
	"github.com/shiningrush/fastflow/pkg/mod"
	"github.com/shiningrush/goevent"
)

var (
	runningTaskCountDesc = prometheus.NewDesc(
		"fastflow_executor_task_running",
		"The count of running task.",
		[]string{"worker_key"}, nil,
	)
	failedTaskCountDesc = prometheus.NewDesc(
		"fastflow_executor_task_failed_total",
		"The count of already failed task.",
		[]string{"worker_key"}, nil,
	)
	failedDagInsDesc = prometheus.NewDesc(
		"fastflow_executor_dag_ins_failed",
		"The count of already failed task.",
		[]string{"worker_key", "dag_ins_id", "business_type", "business_action", "business_id"}, nil,
	)
	successTaskCountDesc = prometheus.NewDesc(
		"fastflow_executor_task_success_total",
		"The count of already failed task.",
		[]string{"worker_key"}, nil,
	)
	completedTaskCountDesc = prometheus.NewDesc(
		"fastflow_executor_task_completed_total",
		"The count of already completed task.",
		[]string{"worker_key"}, nil,
	)

	dispatchInitDagInsElapsedMsDesc = prometheus.NewDesc(
		"fastflow_dispatcher_elapsed_ms",
		"The elapsed time of dispatch init dag instance(ms).",
		[]string{"worker_key"}, nil,
	)
	dispatchInitDagInsFailedCountDesc = prometheus.NewDesc(
		"fastflow_dispatcher_failed_total",
		"The count of dispatch failed.",
		[]string{"worker_key"}, nil,
	)
	parseScheduleDagInsElapsedMsDesc = prometheus.NewDesc(
		"fastflow_parser_parse_scheduled_dag_instance_elapsed_ms",
		"The elapsed time of dispatch init dag instance(ms).",
		[]string{"worker_key"}, nil,
	)
	parseScheduleDagInsFailedCountDesc = prometheus.NewDesc(
		"fastflow_parser_parse_scheduled_dag_instance_failed_total",
		"The count of parse scheduled dag instance failed.",
		[]string{"worker_key"}, nil,
	)
)

// ExecutorCollector
type ExecutorCollector struct {
	rwMutex sync.RWMutex

	RunningTaskCount      int64
	SuccessTaskCount      uint64
	FailedTaskCount       uint64
	FailedTaskDagInsInfos map[string]DagInsInfo
	CompletedTaskCount    uint64

	ParseElapsedMs   int64
	ParseFailedCount int64
}

type DagInsInfo struct {
	BusinessType   string
	BusinessAction string
	BusinessID     string
}

// Topic is goevent's topic
func (c *ExecutorCollector) Topic() []string {
	return []string{event.KeyTaskBegin, event.KeyTaskCompleted, event.KeyParseScheduleDagInsCompleted}
}

// Handle is goevent's handler
func (c *ExecutorCollector) Handle(cxt context.Context, e goevent.Event) {
	if _, ok := e.(*event.TaskBegin); ok {
		atomic.AddInt64(&c.RunningTaskCount, 1)
	}

	if completeEvent, ok := e.(*event.TaskCompleted); ok {
		atomic.AddUint64(&c.CompletedTaskCount, 1)
		if c.RunningTaskCount > 0 {
			atomic.AddInt64(&c.RunningTaskCount, -1)
		}
		switch completeEvent.TaskIns.Status {
		case entity.TaskInstanceStatusFailed:
			atomic.AddUint64(&c.FailedTaskCount, 1)
			c.cacheFailedDagIns(completeEvent)
		case entity.TaskInstanceStatusSuccess:
			atomic.AddUint64(&c.SuccessTaskCount, 1)
		}
	}

	if parseEvent, ok := e.(*event.ParseScheduleDagInsCompleted); ok {
		atomic.StoreInt64(&c.ParseElapsedMs, parseEvent.ElapsedMs)
		if parseEvent.Error != nil {
			atomic.AddInt64(&c.ParseFailedCount, 1)
		}
	}
}

func (c *ExecutorCollector) cacheFailedDagIns(completeEvent *event.TaskCompleted) {
	c.rwMutex.Lock()
	if len(c.FailedTaskDagInsInfos) >= 500 {
		return
	}
	if c.FailedTaskDagInsInfos == nil {
		c.FailedTaskDagInsInfos = map[string]DagInsInfo{}
	}
	dagInsInfo := DagInsInfo{}
	if completeEvent.TaskIns.RelatedDagInstance != nil {
		tags := completeEvent.TaskIns.RelatedDagInstance.Tags
		for _, tag := range tags {
			if tag.Key == "business_type" {
				dagInsInfo.BusinessType = tag.Value
			}
			if tag.Key == "business_action" {
				dagInsInfo.BusinessAction = tag.Value
			}
			if tag.Key == "business_id" {
				dagInsInfo.BusinessID = tag.Value
			}
		}
		c.FailedTaskDagInsInfos[completeEvent.TaskIns.RelatedDagInstance.ID] = dagInsInfo
	}
	c.rwMutex.Unlock()
}

// Describe
func (c *ExecutorCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect
func (c *ExecutorCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		runningTaskCountDesc,
		prometheus.GaugeValue,
		float64(c.RunningTaskCount),
		mod.GetKeeper().WorkerKey(),
	)
	ch <- prometheus.MustNewConstMetric(
		completedTaskCountDesc,
		prometheus.CounterValue,
		float64(c.CompletedTaskCount),
		mod.GetKeeper().WorkerKey(),
	)
	c.pushFailedDagInsInfo(ch)
	ch <- prometheus.MustNewConstMetric(
		failedTaskCountDesc,
		prometheus.CounterValue,
		float64(c.FailedTaskCount),
		mod.GetKeeper().WorkerKey(),
	)
	ch <- prometheus.MustNewConstMetric(
		successTaskCountDesc,
		prometheus.CounterValue,
		float64(c.SuccessTaskCount),
		mod.GetKeeper().WorkerKey(),
	)

	ch <- prometheus.MustNewConstMetric(
		parseScheduleDagInsElapsedMsDesc,
		prometheus.GaugeValue,
		float64(c.ParseElapsedMs),
		mod.GetKeeper().WorkerKey(),
	)
	ch <- prometheus.MustNewConstMetric(
		parseScheduleDagInsFailedCountDesc,
		prometheus.CounterValue,
		float64(c.ParseFailedCount),
		mod.GetKeeper().WorkerKey(),
	)
}

func (c *ExecutorCollector) pushFailedDagInsInfo(ch chan<- prometheus.Metric) {
	c.rwMutex.Lock()
	tempMap := map[string]DagInsInfo{}
	for k, v := range c.FailedTaskDagInsInfos {
		tempMap[k] = v
	}
	c.FailedTaskDagInsInfos = map[string]DagInsInfo{}
	c.rwMutex.Unlock()
	for dagInsID, info := range tempMap {
		ch <- prometheus.MustNewConstMetric(
			failedDagInsDesc,
			prometheus.GaugeValue,
			float64(1),
			mod.GetKeeper().WorkerKey(),
			dagInsID,
			info.BusinessType,
			info.BusinessAction,
			info.BusinessID,
		)
	}
}

// ExecutorCollector
type LeaderCollector struct {
	DispatchElapsedMs   int64
	DispatchFailedCount int64
}

// Topic is goevent's topic
func (c *LeaderCollector) Topic() []string {
	return []string{event.KeyDispatchInitDagInsCompleted}
}

// Handle is goevent's handler
func (c *LeaderCollector) Handle(cxt context.Context, e goevent.Event) {
	if dispatchEvent, ok := e.(*event.DispatchInitDagInsCompleted); ok {
		atomic.StoreInt64(&c.DispatchElapsedMs, dispatchEvent.ElapsedMs)
		if dispatchEvent.Error != nil {
			atomic.AddInt64(&c.DispatchFailedCount, 1)
		}
	}
}

// Describe
func (c *LeaderCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Collect
func (c *LeaderCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		dispatchInitDagInsElapsedMsDesc,
		prometheus.GaugeValue,
		float64(c.DispatchElapsedMs),
		mod.GetKeeper().WorkerKey(),
	)
	ch <- prometheus.MustNewConstMetric(
		dispatchInitDagInsFailedCountDesc,
		prometheus.CounterValue,
		float64(c.DispatchFailedCount),
		mod.GetKeeper().WorkerKey(),
	)
}

// HttpHandler used to handle metrics request
// you can use it like that
//
//	http.Handle("/metrics", exporter.HttpHandler)
//
// because it depend on Keeper, so you should call this function after keeper start
func HttpHandler() http.Handler {
	execCollector := &ExecutorCollector{}
	if err := goevent.Subscribe(execCollector); err != nil {
		panic(err)
	}
	leaderCollector := &LeaderCollector{}
	if err := goevent.Subscribe(leaderCollector); err != nil {
		panic(err)
	}

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(
		execCollector,
		leaderCollector,
		// Add the standard process and Go metrics to the custom registry.
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
