package maas

import (
	"fmt"
	"time"

	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/alecthomas/kingpin/v2"
)

type Advertiser interface {
	Flags(a *kingpin.Application)
	Advertise(address string, port int, labels Labeler) error
}

type HealthCheck struct {
	frequency  time.Duration
	timeout    time.Duration
	deregister time.Duration
}

type Consul struct {
	config      *consul.Config
	client      *consul.Client
	healthCheck HealthCheck
}

func NewConsul(options ...func(*Consul)) *Consul {
	c := &Consul{
		config: consul.DefaultConfig(),
	}

	c.apply(options)

	cl, err := consul.NewClient(c.config)

	if err != nil {
		log.Errorf("unable to create client: %s", err)
		return nil
	}

	c.client = cl

	return c
}

func (c *Consul) Flags(a *kingpin.Application) {
	a.Flag("healthcheck.frequency", "Frequency to perform healthcheck").Default("10s").DurationVar(&c.healthCheck.frequency)
	a.Flag("healthcheck.timeout", "Timeout for healthcheck").Default("5s").DurationVar(&c.healthCheck.timeout)
	a.Flag("healthcheck.deregister",
		"critical duration before service is deregistered").
		Default("5m").DurationVar(&c.healthCheck.deregister)
}

func (c *Consul) Advertise(address string, port int, labels Labeler) error {
	return c.client.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		Name:    fmt.Sprintf("%s:%d", address, port),
		Address: address,
		Port:    port,
		Tags:    c.tags(labels),
		Check: &consul.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", address, port),
			Timeout:                        c.healthCheck.timeout.String(),
			Interval:                       c.healthCheck.frequency.String(),
			DeregisterCriticalServiceAfter: c.healthCheck.deregister.String(),
		},
	})
}

func (c *Consul) tags(l Labeler) []string {
	t := make([]string, 0, len(l.Labels()))

	for k, v := range l.Labels() {
		t = append(t, fmt.Sprintf("%s=%s", k, v))
	}

	return t
}

func (c *Consul) apply(options []func(*Consul)) {
	for _, option := range options {
		option(c)
	}
}

func WithConfig(cfg *consul.Config) func(*Consul) {
	return func(c *Consul) {
		c.config = cfg
	}
}
