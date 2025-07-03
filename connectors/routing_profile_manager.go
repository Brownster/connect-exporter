// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file defines the RoutingProfileManager, a component that periodically refreshes
// the list of routing profiles from AWS Connect and provides a cached, thread-safe way for
// collectors to access routing profile data. This avoids redundant API calls and improves
// the overall performance of the exporter.
package connectors

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/sirupsen/logrus"
)

// RoutingProfileLister defines the interface for listing routing profiles. It is implemented by
// AWSConnectConnector and allows the RoutingProfileManager to be tested independently.
type RoutingProfileLister interface {
	ListRoutingProfiles(ctx context.Context, input *connect.ListRoutingProfilesInput) (*connect.ListRoutingProfilesOutput, error)
	GetInstanceID() string
}

// RoutingProfileManager periodically refreshes the list of routing profiles from AWS Connect and
// provides cached access for collectors.
type RoutingProfileManager struct {
	lister          RoutingProfileLister
	refreshInterval time.Duration

	mu              sync.RWMutex
	routingProfiles []RoutingProfile
	stopCh          chan struct{}
}

// NewRoutingProfileManager creates and initializes a new RoutingProfileManager.
func NewRoutingProfileManager(lister RoutingProfileLister, interval time.Duration) *RoutingProfileManager {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &RoutingProfileManager{
		lister:          lister,
		refreshInterval: interval,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the periodic refresh of routing profiles. This method should be called
// once the RoutingProfileManager is created.
func (m *RoutingProfileManager) Start() {
	logrus.WithField("interval", m.refreshInterval).Info("Starting routing profile manager")
	
	// Initial fetch
	if err := m.refresh(); err != nil {
		logrus.WithError(err).Error("Failed initial routing profile refresh")
	}

	// Start periodic refresh
	go func() {
		ticker := time.NewTicker(m.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := m.refresh(); err != nil {
					logrus.WithError(err).Error("Failed to refresh routing profiles")
				}
			case <-m.stopCh:
				logrus.Info("Stopping routing profile manager")
				return
			}
		}
	}()
}

// Stop gracefully stops the RoutingProfileManager.
func (m *RoutingProfileManager) Stop() {
	close(m.stopCh)
}

// GetRoutingProfiles returns a copy of the current routing profiles list.
func (m *RoutingProfileManager) GetRoutingProfiles() []RoutingProfile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]RoutingProfile, len(m.routingProfiles))
	copy(result, m.routingProfiles)
	return result
}

// GetRoutingProfileMap returns a map of routing profile ID to RoutingProfile for quick lookups.
func (m *RoutingProfileManager) GetRoutingProfileMap() map[string]RoutingProfile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]RoutingProfile, len(m.routingProfiles))
	for _, rp := range m.routingProfiles {
		result[rp.ID] = rp
	}
	return result
}

// refresh fetches the latest routing profiles from AWS Connect and updates the cache.
func (m *RoutingProfileManager) refresh() error {
	logrus.Debug("Refreshing routing profiles")

	ctx := context.Background()
	instanceID := m.lister.GetInstanceID()

	var allRoutingProfiles []RoutingProfile
	var nextToken *string

	for {
		input := &connect.ListRoutingProfilesInput{
			InstanceId: &instanceID,
			NextToken:  nextToken,
			MaxResults: aws.Int32(1000),
		}

		output, err := m.lister.ListRoutingProfiles(ctx, input)
		if err != nil {
			return err
		}

		for _, rp := range output.RoutingProfileSummaryList {
			if rp.Id != nil && rp.Name != nil {
				routingProfile := RoutingProfile{
					ID:   aws.ToString(rp.Id),
					Name: aws.ToString(rp.Name),
				}
				allRoutingProfiles = append(allRoutingProfiles, routingProfile)
			}
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	m.mu.Lock()
	m.routingProfiles = allRoutingProfiles
	m.mu.Unlock()

	logrus.WithField("count", len(allRoutingProfiles)).Info("Refreshed routing profiles")
	return nil
}