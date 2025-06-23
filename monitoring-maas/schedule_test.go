package maas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ScheduleTestSuite struct {
	suite.Suite
}

func (s *ScheduleTestSuite) TestDefaultSchedule() {
	const frequency = 10 * time.Second

	sc := NewSchedule()
	s.Equal(sc.frequency, frequency)
	s.Equal(sc.timeout, time.Second)
	s.True(sc.isEnabled)
}

func (s *ScheduleTestSuite) TestCustomSchedule() {
	sc := NewSchedule(
		WithFrequency(time.Hour),
		WithTimeout(time.Minute),
		Disabled(),
	)
	s.Equal(sc.frequency, time.Hour)
	s.Equal(sc.timeout, time.Minute)
	s.False(sc.isEnabled)
}

func TestScheduleTestSuite(t *testing.T) {
	suite.Run(t, new(ScheduleTestSuite))
}
