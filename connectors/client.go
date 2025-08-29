// Package connectors provides the interfaces and implementations for connecting to
// external services like AWS Connect.
//
// This file contains the client-side logic for interacting with the AWS Connect API.
// It wraps the AWS SDK calls in methods that are easy to use and test. By isolating
// the API interaction, this file provides a clear separation of concerns and makes
// the rest of the connector code more readable and maintainable.
package connectors

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/connect"
	"github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/sirupsen/logrus"
)

// ListQueues retrieves all queues for the AWS Connect instance. It handles
// pagination automatically, collecting all queues into a single slice.
func (c *AWSConnectConnector) ListQueues(ctx context.Context) ([]Queue, error) {
	var queues []Queue
	var nextToken *string

	for {
		instanceID := c.GetInstanceID()
		output, err := c.client.ListQueues(ctx, &connect.ListQueuesInput{
			InstanceId: &instanceID,
			NextToken:  nextToken,
			MaxResults: aws.Int32(1000),
		})
		if err != nil {
			return nil, fmt.Errorf("error listing queues: %w", err)
		}

		for _, q := range output.QueueSummaryList {
			queues = append(queues, Queue{
				ID:   aws.ToString(q.Id),
				Name: aws.ToString(q.Name),
				Type: string(q.QueueType),
			})
		}

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	logrus.WithFields(logrus.Fields{
		"queue_count":  len(queues),
		"instance_arn": c.instanceARN,
		"region":       c.region,
	}).Debug("Successfully listed queues")

	return queues, nil
}

// GetCurrentMetricData retrieves current metrics for the specified filters.
// This is a direct wrapper around the AWS SDK's GetCurrentMetricData method.
func (c *AWSConnectConnector) GetCurrentMetricData(ctx context.Context, input *connect.GetCurrentMetricDataInput) (*connect.GetCurrentMetricDataOutput, error) {
	output, err := c.client.GetCurrentMetricData(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting current metric data: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"metric_count": len(output.MetricResults),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved current metric data")

	return output, nil
}

// GetMetricDataV2 retrieves historical metrics using the V2 API. This is a
// direct wrapper around the AWS SDK's GetMetricDataV2 method.
func (c *AWSConnectConnector) GetMetricDataV2(ctx context.Context, input *connect.GetMetricDataV2Input) (*connect.GetMetricDataV2Output, error) {
	output, err := c.client.GetMetricDataV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error getting metric data v2: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"metric_count": len(output.MetricResults),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved metric data v2")

	return output, nil
}

// GetCurrentUserData retrieves current user data, including agent status.
// It handles pagination and retries to ensure all user data is fetched.
func (c *AWSConnectConnector) GetCurrentUserData(ctx context.Context, input *connect.GetCurrentUserDataInput) (*connect.GetCurrentUserDataOutput, error) {
	instanceID := input.InstanceId
	var users []types.UserData
	var count int64
	var nextToken *string
	for {
		out, err := c.client.GetCurrentUserData(ctx, &connect.GetCurrentUserDataInput{
			InstanceId: instanceID,
			Filters:    input.Filters,
			MaxResults: input.MaxResults,
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("error getting current user data: %w", err)
		}
		users = append(users, out.UserDataList...)
		if out.ApproximateTotalCount != nil {
			count = *out.ApproximateTotalCount
		}
		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	logrus.WithFields(logrus.Fields{
		"user_count":   len(users),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully retrieved current user data")

	return &connect.GetCurrentUserDataOutput{UserDataList: users, ApproximateTotalCount: aws.Int64(count)}, nil
}

// ListAgentStatuses retrieves agent status information from the AWS Connect instance.
func (c *AWSConnectConnector) ListAgentStatuses(ctx context.Context, input *connect.ListAgentStatusesInput) (*connect.ListAgentStatusesOutput, error) {
	output, err := c.client.ListAgentStatuses(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error listing agent statuses: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"status_count": len(output.AgentStatusSummaryList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully listed agent statuses")

	return output, nil
}

// ListRoutingProfileQueues retrieves the queue IDs associated with a routing profile.
func (c *AWSConnectConnector) ListRoutingProfileQueues(ctx context.Context, input *connect.ListRoutingProfileQueuesInput) (*connect.ListRoutingProfileQueuesOutput, error) {
	output, err := c.client.ListRoutingProfileQueues(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error listing routing profile queues: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"queue_count":  len(output.RoutingProfileQueueConfigSummaryList),
		"instance_arn": c.instanceARN,
	}).Debug("Successfully listed routing profile queues")

	return output, nil
}
