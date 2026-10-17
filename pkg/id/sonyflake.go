package id

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

var errSonyflakeInitialization = errors.New("create sonyflake failed")

// Sonyflake wraps an ID generator and its initialization error.
type Sonyflake struct {
	ops   SonyflakeOptions
	sf    *sonyflake.Sonyflake
	Error error
}

// NewSonyflake 可以根据 id 生成唯一编码（你需要确保 id 是唯一的）。
func NewSonyflake(options ...func(*SonyflakeOptions)) *Sonyflake {
	ops := getSonyflakeOptionsOrSetDefault(nil)
	for _, f := range options {
		f(ops)
	}
	sf := &Sonyflake{
		ops: *ops,
	}
	st := sonyflake.Settings{
		StartTime: ops.startTime,
	}
	if ops.machineId > 0 {
		st.MachineID = func() (uint16, error) {
			return ops.machineId, nil
		}
	}
	ins := sonyflake.NewSonyflake(st)
	if ins == nil {
		sf.Error = errSonyflakeInitialization
		return sf
	}
	_, err := ins.NextID()
	if err != nil {
		sf.Error = fmt.Errorf("initialize sonyflake: %w", err)
	}
	sf.sf = ins
	return sf
}

// Id returns an ID, or zero if initialization failed.
func (s *Sonyflake) Id(ctx context.Context) uint64 {
	if s.Error != nil {
		return 0
	}
	id, err := s.sf.NextID()
	if err == nil {
		return id
	}

	sleep := 1
	for {
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		id, err = s.sf.NextID()
		if err == nil {
			return id
		}
		sleep *= 2
	}
}
