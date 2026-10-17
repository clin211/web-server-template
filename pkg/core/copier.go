package core

import (
	"errors"
	"time"

	"github.com/jinzhu/copier"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errSourceTypeMismatch = errors.New("source type not matching")

// TypeConverters 定义时间类型转换器，用于 copier 的深度拷贝。
func TypeConverters() []copier.TypeConverter {
	return []copier.TypeConverter{
		{
			SrcType: time.Time{},
			DstType: &timestamppb.Timestamp{},
			Fn: func(src any) (any, error) {
				s, ok := src.(time.Time)
				if !ok {
					return nil, errSourceTypeMismatch
				}
				return timestamppb.New(s), nil
			},
		},
		{
			SrcType: &timestamppb.Timestamp{},
			DstType: time.Time{},
			Fn: func(src any) (any, error) {
				s, ok := src.(*timestamppb.Timestamp)
				if !ok {
					return nil, errSourceTypeMismatch
				}
				return s.AsTime(), nil
			},
		},
		{
			SrcType: time.Time{},
			DstType: int64(0),
			Fn: func(src any) (any, error) {
				s, ok := src.(time.Time)
				if !ok {
					return nil, errSourceTypeMismatch
				}
				return s.Unix(), nil
			},
		},
		{
			SrcType: int64(0),
			DstType: time.Time{},
			Fn: func(src any) (any, error) {
				s, ok := src.(int64)
				if !ok {
					return nil, errSourceTypeMismatch
				}
				return time.Unix(s, 0), nil
			},
		},
	}
}

func CopyWithConverters(to any, from any, converters ...copier.TypeConverter) error {
	converters = append(TypeConverters(), converters...)
	return copier.CopyWithOption(to, from, copier.Option{IgnoreEmpty: true, DeepCopy: true, Converters: converters})
}

func Copy(to any, from any) error {
	return copier.Copy(to, from)
}
