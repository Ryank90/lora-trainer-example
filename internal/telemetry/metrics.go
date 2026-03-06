package telemetry

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	JobsCreated       *prometheus.CounterVec
	JobsCompleted     *prometheus.CounterVec
	JobsFailed        *prometheus.CounterVec
	JobTotalDuration  *prometheus.HistogramVec
	JobPhaseDuration  *prometheus.HistogramVec
	ActiveInstances   *prometheus.GaugeVec
	InstanceCost      *prometheus.CounterVec
	QueueDepth        prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		JobsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lora_trainer_jobs_created_total",
				Help: "Total number of training jobs created",
			},
			[]string{"model_type"},
		),
		JobsCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lora_trainer_jobs_completed_total",
				Help: "Total number of training jobs completed",
			},
			[]string{"model_type", "provider"},
		),
		JobsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lora_trainer_jobs_failed_total",
				Help: "Total number of training jobs failed",
			},
			[]string{"model_type", "provider", "phase"},
		),
		JobTotalDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "lora_trainer_job_total_duration_seconds",
				Help:    "Total job duration from creation to completion",
				Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hrs
			},
			[]string{"model_type", "provider", "gpu_type"},
		),
		JobPhaseDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "lora_trainer_job_phase_duration_seconds",
				Help:    "Duration of each job phase",
				Buckets: prometheus.ExponentialBuckets(1, 2, 15), // 1s to ~9hrs
			},
			[]string{"phase"},
		),
		ActiveInstances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "lora_trainer_active_instances",
				Help: "Number of currently active GPU instances",
			},
			[]string{"provider"},
		),
		InstanceCost: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "lora_trainer_instance_cost_dollars_total",
				Help: "Total instance cost in dollars",
			},
			[]string{"provider", "gpu_type"},
		),
		QueueDepth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "lora_trainer_queue_depth",
				Help: "Number of jobs in the pending queue",
			},
		),
	}

	reg.MustRegister(
		m.JobsCreated,
		m.JobsCompleted,
		m.JobsFailed,
		m.JobTotalDuration,
		m.JobPhaseDuration,
		m.ActiveInstances,
		m.InstanceCost,
		m.QueueDepth,
	)

	return m
}
