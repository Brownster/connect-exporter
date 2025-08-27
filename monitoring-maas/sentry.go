package maas

import (
	"errors"
	"fmt"

	"github.com/onrik/logrus/sentry"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

var SentryDSN string

func RegisterSentryHook(l Labeler) error {
	if SentryDSN == "" {
		return errors.New("no DSN provided")
	}

	labels := l.Labels()

	if _, ok := labels["environment"]; !ok {
		return errors.New("no environment provided")
	}

	if _, ok := labels["exporter"]; !ok {
		return errors.New("no exporter provided")
	}

	if _, ok := labels["fqdn"]; !ok {
		return errors.New("no fqdn provided")
	}

	h, err := sentry.NewHook(sentry.Options{
		Dsn:         SentryDSN,
		Environment: labels["environment"],
		ServerName:  labels["fqdn"],
		Release:     fmt.Sprintf("%s_exporter@%s", labels["exporter"], version.Version),
	})

	if err != nil {
		return err
	}

	log.AddHook(h)

	log.Infof("registered sentry hook with DSN: %s", SentryDSN)

	return nil
}
