package repository

import (
	"context"
	"core/domain/entity"
)

// UserEventLogRepository 用户事件日志仓储接口
type UserEventLogRepository interface {
	// SaveEventLog 保存单个事件日志（同步）
	SaveEventLog(ctx context.Context, log *entity.UserEventLog) error

	// SaveEventLogs 批量保存事件日志（同步）
	SaveEventLogs(ctx context.Context, logs []*entity.UserEventLog) error

	// SaveEventLogAsync 异步保存单个事件日志（非阻塞）
	SaveEventLogAsync(log *entity.UserEventLog)

	// SaveEventLogsAsync 异步批量保存事件日志（非阻塞）
	SaveEventLogsAsync(logs []*entity.UserEventLog)

	// FindEventLogsByUser 查找用户的事件日志（分页）
	FindEventLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.UserEventLog, error)

	// FindEventLogsByUserAndType 查找用户指定类型的事件日志（分页）
	FindEventLogsByUserAndType(ctx context.Context, userID, eventType string, limit, offset int) ([]*entity.UserEventLog, error)

	// FindEventLogsByTimeRange 按时间范围查找事件日志（分页）
	FindEventLogsByTimeRange(ctx context.Context, startTime, endTime int64, limit, offset int) ([]*entity.UserEventLog, error)

	// FindEventLogsByUserAndTimeRange 查找用户在指定时间范围内的事件日志（分页）
	FindEventLogsByUserAndTimeRange(ctx context.Context, userID string, startTime, endTime int64, limit, offset int) ([]*entity.UserEventLog, error)

	// Close 关闭异步写入通道（优雅关闭）
	Close() error
}
