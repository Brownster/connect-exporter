package collectors

import (
	"reflect"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	maas "github.com/sabio-engineering-product/connect-exporter/monitoring-maas"
)

func metricValueType(m maas.Metric) prometheus.ValueType {
	v := reflect.ValueOf(&m).Elem()
	f := v.FieldByName("valueType")
	return *(*prometheus.ValueType)(unsafe.Pointer(f.UnsafeAddr()))
}

func metricLabels(m maas.Metric) []string {
	v := reflect.ValueOf(&m).Elem()
	f := v.FieldByName("labels")
	return *(*[]string)(unsafe.Pointer(f.UnsafeAddr()))
}

func metricName(m maas.Metric) string {
	v := reflect.ValueOf(&m).Elem()
	f := v.FieldByName("name")
	return *(*string)(unsafe.Pointer(f.UnsafeAddr()))
}
