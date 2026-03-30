package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sachinggsingh/firstclick/internal/model"
	"github.com/sachinggsingh/firstclick/internal/store"
)

func TestConcurrentBooking(t *testing.T) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at localhost:6379: %v", err)
	}

	rdb := store.NewRedisStore(client)
	svc := NewBookingService(rdb, nil)

	const numGoroutines = 10000 // 10K users

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			_, err := svc.Book(ctx, &model.Booking{
				MovieID: "screen-1",
				SeatID:  "A1",
				UserID:  uuid.New().String(),
			})
			if err == nil {
				successes.Add(1)
			} else {
				failures.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 success, got %d", got)
	}
	if got := failures.Load(); got != int64(numGoroutines-1) {
		t.Errorf("expected %d failures, got %d", numGoroutines-1, got)
	}
}
