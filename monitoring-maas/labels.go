package maas

import (
	"net"

	"github.com/alecthomas/kingpin/v2"
)

type Labeler interface {
	Labels() map[string]string
	Flags(a *kingpin.Application)
}

type DefaultLabels struct {
	fqdn        string
	exporter    string
	environment string
	location    string
	country     string
	ipAddress   net.IP
}

func NewDefaultLabels() *DefaultLabels {
	return &DefaultLabels{}
}

func (l *DefaultLabels) Labels() map[string]string {
	return map[string]string{
		"fqdn":        l.fqdn,
		"exporter":    l.exporter,
		"environment": l.environment,
		"location":    l.location,
		"country":     l.country,
		"ip_address":  l.ipAddress.String(),
	}
}

func (l *DefaultLabels) Flags(a *kingpin.Application) {
	l.exporter = a.Name
	a.Flag("labels.environment", "environment target label").Default("production").StringVar(&l.environment)
	a.Flag("labels.fqdn", "FDQN target label").Required().StringVar(&l.fqdn)
	a.Flag("labels.location", "location target label").Required().StringVar(&l.location)
	a.Flag("labels.country", "2 letter country code target label").Required().StringVar(&l.country)
	a.Flag("labels.ip-address", "IP address target label").Required().IPVar(&l.ipAddress)
}
