package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sachinggsingh/firstclick/internal/logger"
	"github.com/sachinggsingh/firstclick/internal/model"
)

type MemoryStore struct {
	Bookings map[string]model.Booking
}
type RedisStore struct {
	rdb    *redis.Client
	logger *logger.Logger
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb, logger: logger.NewLogger()}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		Bookings: map[string]model.Booking{},
	}
}

func (r *RedisStore) sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}

func (r *RedisStore) Hold(booking model.Booking) (model.Booking, error) {
	if booking.MovieID == "" || booking.SeatID == "" {
		return model.Booking{}, fmt.Errorf("movie_id and seat_id are required")
	}

	now := time.Now()
	id := uuid.New().String()
	ctx := context.Background()

	seatKey := fmt.Sprintf("seat:%s:%s", booking.MovieID, booking.SeatID)
	defaultHold := 20 * time.Second

	booking.ID = id
	booking.Status = model.SeatHold
	booking.ExpiresAt = now.Add(defaultHold)

	val, err := json.Marshal(booking)
	if err != nil {
		return model.Booking{}, err
	}

	ok, err := r.rdb.SetNX(ctx, seatKey, val, defaultHold).Result()
	if err != nil {
		return model.Booking{}, err
	}
	if !ok {
		// Seat already held/confirmed (key exists)
		return model.Booking{}, nil
	}

	// Map session -> seat key so confirm/release can locate the seat booking.
	// TTL mirrors the hold TTL; confirm will persist both keys.
	if err := r.rdb.Set(ctx, r.sessionKey(id), seatKey, defaultHold).Err(); err != nil {
		_ = r.rdb.Del(ctx, seatKey).Err()
		return model.Booking{}, err
	}

	return booking, nil
}

func ParseSession(val string) (model.Booking, error) {
	var data model.Booking
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return model.Booking{}, err
	}
	return data, nil
}

func (r *RedisStore) GetSession(ctx context.Context, sessionID string, userID string) (model.Booking, string, error) {
	seatKey, err := r.rdb.Get(ctx, r.sessionKey(sessionID)).Result()
	if err != nil {
		return model.Booking{}, "", err
	}
	val, err := r.rdb.Get(ctx, seatKey).Result()
	if err != nil {
		return model.Booking{}, "", err
	}
	session, err := ParseSession(val)
	if err != nil {
		return model.Booking{}, "", err
	}
	if session.UserID != userID {
		return model.Booking{}, "", fmt.Errorf("session does not belong to this user")
	}
	return session, seatKey, nil
}

func (r *RedisStore) Book(booking model.Booking) (model.Booking, error) {
	session, err := r.Hold(booking)
	if err != nil {
		return model.Booking{}, err
	}
	logger.NewLogger().Info("Session Booked", "Session_id", booking.ID)
	return session, nil
}

func (r *RedisStore) GetBookedSeats(movieID string) []model.Booking {
	pattern := fmt.Sprintf("seat:%s:*", movieID)
	var sessions []model.Booking
	ctx := context.Background()

	item := r.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for item.Next(ctx) {
		val, err := r.rdb.Get(ctx, item.Val()).Result()
		if err != nil {
			continue
		}
		session, err := ParseSession(val)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions
}
func (r *RedisStore) Release(ctx context.Context, sessionID string, userID string) error {
	_, sk, err := r.GetSession(ctx, sessionID, userID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Can't get the session id and user id", userID, sessionID)
		}
		// If the session doesn't exist or doesn't belong to the user, treat as idempotent.
		return nil
	}
	_ = r.rdb.Del(ctx, sk, r.sessionKey(sessionID)).Err()
	return nil
}

func (r *RedisStore) Confirm(ctx context.Context, sessionID string, userID string) (model.Booking, error) {
	session, sk, err := r.GetSession(ctx, sessionID, userID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Error in getting the session", session, err)
		}
		return model.Booking{}, err
	}

	session.Status = model.SeatConfirm
	val, err := json.Marshal(session)
	if err != nil {
		return model.Booking{}, err
	}

	// Persist both seat booking and session mapping (no TTL).
	if err := r.rdb.Set(ctx, sk, val, 0).Err(); err != nil {
		return model.Booking{}, err
	}
	if err := r.rdb.Set(ctx, r.sessionKey(sessionID), sk, 0).Err(); err != nil {
		return model.Booking{}, err
	}

	return session, nil

}
