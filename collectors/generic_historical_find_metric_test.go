package collectors

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/sabio-engineering-product/connect-exporter/connectors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindMetricDefinition_NoFilters(t *testing.T) {
	s := NewGenericHistoricalScraper()
	defs := []connectors.MetricDefinition{
		{AWSName: "AVG_HANDLE_TIME", PrometheusName: "aht"},
	}
	def, ok := s.findMetricDefinition("AVG_HANDLE_TIME", nil, defs)
	assert.True(t, ok)
	assert.Equal(t, "aht", def.PrometheusName)
}

func TestFindMetricDefinition_WithMatchingFilter(t *testing.T) {
	s := NewGenericHistoricalScraper()
	defs := []connectors.MetricDefinition{
		{
			AWSName:        "CONTACTS_HANDLED",
			PrometheusName: "contacts_handled_inbound",
			MetricFilters:  []connectors.MetricFilter{{Key: "DIRECTION", Values: []string{"INBOUND"}}},
		},
	}
	filters := []types.MetricFilterV2{{MetricFilterKey: aws.String("DIRECTION"), MetricFilterValues: []string{"INBOUND"}}}
	def, ok := s.findMetricDefinition("CONTACTS_HANDLED", filters, defs)
	assert.True(t, ok)
	assert.Equal(t, "contacts_handled_inbound", def.PrometheusName)
}

func TestFindMetricDefinition_Mismatch(t *testing.T) {
	s := NewGenericHistoricalScraper()
	defs := []connectors.MetricDefinition{
		{
			AWSName:        "CONTACTS_HANDLED",
			PrometheusName: "contacts_handled_outbound",
			MetricFilters:  []connectors.MetricFilter{{Key: "DIRECTION", Values: []string{"OUTBOUND"}}},
		},
	}
	filters := []types.MetricFilterV2{{MetricFilterKey: aws.String("DIRECTION"), MetricFilterValues: []string{"INBOUND"}}}
	_, ok := s.findMetricDefinition("CONTACTS_HANDLED", filters, defs)
	assert.False(t, ok)
}
