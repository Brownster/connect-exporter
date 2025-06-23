package maas

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
)

type Connector interface {
	Connect() error
	Execute(command interface{}) (interface{}, error)
	Flags(a *kingpin.Application)
}

type UnableToConnectError struct {
	Err error
}

func (e UnableToConnectError) Error() string {
	return fmt.Sprintf("unable to connect: %s", e.Err)
}
