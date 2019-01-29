package service

import (
	"github.com/integration-system/isp-lib/metric"
	"github.com/rcrowley/go-metrics"
	"strconv"
	"sync"
	"time"
)

const (
	defaultSampleSize = 2048
)

var (
	mh *metricHolder
)

type metricHolder struct {
	methodHistograms   map[string]metrics.Histogram
	methodLock         sync.RWMutex
	statusCounters     map[int]metrics.Counter
	statusLock         sync.RWMutex
	routerResponseTime metrics.Histogram
	responseTime       metrics.Histogram
}

func (mh *metricHolder) UpdateMethodResponseTime(uri string, time time.Duration) {
	mh.getOrRegisterHistogram(uri).Update(int64(time))
}

func (mh *metricHolder) UpdateResponseTime(time time.Duration) {
	mh.responseTime.Update(int64(time))
}

func (mh *metricHolder) UpdateRouterResponseTime(time time.Duration) {
	mh.routerResponseTime.Update(int64(time))
}

func (mh *metricHolder) UpdateStatusCounter(status int) {
	mh.getOrRegisterCounter(status).Inc(1)
}

func (mh *metricHolder) getOrRegisterHistogram(uri string) metrics.Histogram {
	mh.methodLock.RLock()
	histogram, ok := mh.methodHistograms[uri]
	mh.methodLock.RUnlock()
	if ok {
		return histogram
	}

	mh.methodLock.Lock()
	defer mh.methodLock.Unlock()
	if d, ok := mh.methodHistograms[uri]; ok {
		return d
	}
	histogram = metrics.GetOrRegisterHistogram(
		"http.response.time_"+uri,
		metric.GetRegistry(),
		metrics.NewUniformSample(defaultSampleSize),
	)
	mh.methodHistograms[uri] = histogram
	return histogram
}

func (mh *metricHolder) getOrRegisterCounter(status int) metrics.Counter {
	mh.statusLock.RLock()
	d, ok := mh.statusCounters[status]
	mh.statusLock.RUnlock()
	if ok {
		return d
	}

	mh.statusLock.Lock()
	defer mh.statusLock.Unlock()
	if d, ok := mh.statusCounters[status]; ok {
		return d
	}
	d = metrics.GetOrRegisterCounter("http.response.count."+strconv.Itoa(status), metric.GetRegistry())
	mh.statusCounters[status] = d
	return d
}

func GetMetrics() *metricHolder {
	return mh
}

func InitMetrics() {
	if mh == nil {
		mh = &metricHolder{
			methodHistograms: make(map[string]metrics.Histogram),
			statusCounters:   make(map[int]metrics.Counter),
			responseTime: metrics.GetOrRegisterHistogram(
				"http.response.time", metric.GetRegistry(), metrics.NewUniformSample(defaultSampleSize),
			),
			routerResponseTime: metrics.GetOrRegisterHistogram(
				"grpc.router.response.time", metric.GetRegistry(), metrics.NewUniformSample(defaultSampleSize),
			),
		}
	}
}
