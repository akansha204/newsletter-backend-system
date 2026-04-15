package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	EmailsSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "emails_sent_total",
			Help: "Total number of emails sent successfully",
		},
	)

	EmailsFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "emails_failed_total",
			Help: "Total number of failed email sends",
		},
	)

	EmailProcessingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "email_processing_duration_seconds",
			Help:    "Time taken to process email sending",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func Init() {
	prometheus.MustRegister(EmailsSent)
	prometheus.MustRegister(EmailsFailed)
	prometheus.MustRegister(EmailProcessingDuration)
}
