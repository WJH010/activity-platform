package stock

import (
	"context"
	"fmt"
	"time"

	rd "event-platform/internal/redis"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	stockKeyPrefix = "event:stock:"
)

func stockKey(eventID int) string {
	return fmt.Sprintf("%s%d", stockKeyPrefix, eventID)
}

var (
	// decrScript 预扣减脚本，用于检查名额是否充足并扣减
	decrScript = redis.NewScript(`
		local current = tonumber(redis.call('GET', KEYS[1]))
		if current == nil then
			return -1
		end
		if current <= 0 then
			return 0
		end
		redis.call('DECR', KEYS[1])
		return 1
	`)
	// incrScript 回补脚本，用于回补名额
	incrScript = redis.NewScript(`
		local current = tonumber(redis.call('GET', KEYS[1]))
		if current == nil then
			return -1
		end
		redis.call('INCR', KEYS[1])
		return 1
	`)
)

const (
	DecrResultSuccess = 1  // 预扣成功
	DecrResultNoLimit = -1 // 无限制
	DecrResultSoldOut = 0  // 已满
	IncrResultSuccess = 1  // 回补成功
	IncrResultNoLimit = -1 // 无限制
)

type StockService struct{}

func NewStockService() *StockService {
	return &StockService{}
}

func (s *StockService) getClient() *redis.Client {
	return rd.GetClient()
}

// Decr 预扣减库存
func (s *StockService) Decr(ctx context.Context, eventID int) (int, error) {
	rdb := s.getClient()
	if rdb == nil {
		// Redis不可用 → 降级为不限，交给DB兜底
		// 返回 NoLimit ，表示"不限制"，让请求继续走到 DB 层，由 FOR UPDATE 行锁来保证正确性。
		// 这样系统在 Redis 故障时仍然可用，只是性能会下降（所有压力落到 DB）。
		return DecrResultNoLimit, nil
	}
	result, err := decrScript.Run(ctx, rdb, []string{stockKey(eventID)}).Int()
	if err != nil {
		logrus.Warnf("Redis库存预扣失败[eventID=%d]: %v, 降级为不限人数", eventID, err)
		return DecrResultNoLimit, nil
	}
	return result, nil
}

// Incr 回补库存
func (s *StockService) Incr(ctx context.Context, eventID int) (int, error) {
	rdb := s.getClient()
	if rdb == nil {
		return IncrResultNoLimit, nil
	}
	result, err := incrScript.Run(ctx, rdb, []string{stockKey(eventID)}).Int()
	if err != nil {
		logrus.Warnf("Redis库存回补失败[eventID=%d]: %v", eventID, err)
		return IncrResultNoLimit, nil
	}
	return result, nil
}

// Init 初始化活动名额
func (s *StockService) Init(ctx context.Context, eventID int, maxRegistrants int, currentRegistrants int) error {
	rdb := s.getClient()
	if rdb == nil {
		return nil
	}
	if maxRegistrants <= 0 {
		rdb.Del(ctx, stockKey(eventID))
		return nil
	}
	remaining := maxRegistrants - currentRegistrants
	if remaining < 0 {
		remaining = 0
	}
	return rdb.Set(ctx, stockKey(eventID), remaining, 0).Err()
}

// InitWithTTL 初始化活动名额，设置过期时间
func (s *StockService) InitWithTTL(ctx context.Context, eventID int, maxRegistrants int, currentRegistrants int, registrationEndTime time.Time) error {
	rdb := s.getClient()
	if rdb == nil {
		return nil
	}
	if maxRegistrants <= 0 {
		rdb.Del(ctx, stockKey(eventID))
		return nil
	}
	remaining := maxRegistrants - currentRegistrants
	if remaining < 0 {
		remaining = 0
	}
	ttl := time.Until(registrationEndTime) + time.Hour
	if ttl <= 0 {
		ttl = time.Hour
	}
	return rdb.Set(ctx, stockKey(eventID), remaining, ttl).Err()
}

// Delete 删除活动名额
func (s *StockService) Delete(ctx context.Context, eventID int) error {
	rdb := s.getClient()
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, stockKey(eventID)).Err()
}

// Get 获取活动名额
func (s *StockService) Get(ctx context.Context, eventID int) (int, bool, error) {
	rdb := s.getClient()
	if rdb == nil {
		return 0, false, nil
	}
	val, err := rdb.Get(ctx, stockKey(eventID)).Int()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return val, true, nil
}
