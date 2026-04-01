package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/sachinggsingh/firstclick/internal/logger"
	"github.com/sachinggsingh/firstclick/internal/model"
	"github.com/sachinggsingh/firstclick/internal/store"
)

// BookingService provides booking operations backed by Redis.
// It implements model.BookingSeat.
type BookingService struct {
	mu     sync.Mutex
	rdb    *store.RedisStore
	logger *logger.Logger
}

func NewBookingService(rdb *store.RedisStore, log *logger.Logger) *BookingService {
	return &BookingService{rdb: rdb, logger: log}
}

func (bs *BookingService) Book(ctx context.Context, booking *model.Booking) (model.SeatStatus, error) {
	if booking == nil {
		return model.SeatAvailable, fmt.Errorf("booking is nil")
	}
	if bs.rdb == nil {
		return model.SeatAvailable, fmt.Errorf("redis store is nil")
	}
	if booking.SeatID == "" {
		return model.SeatAvailable, fmt.Errorf("seat_id is required")
	}

	// Service-level lock prevents any accidental non-atomic usage of the booking input,
	// but the real "only one winner" rule is enforced by Redis NX in RedisStore.Hold.
	bs.mu.Lock()
	defer bs.mu.Unlock()

	session, err := bs.rdb.Hold(*booking)
	if err != nil {
		return model.SeatAvailable, err
	}

	// RedisStore.Hold returns an empty Booking (ID == "") when the seat key already exists.
	if session.ID == "" {
		if bs.logger != nil {
			bs.logger.Warn(
				"Seat already held",
				"seat_id", booking.SeatID,
				"movie_id", booking.MovieID,
				"user_id", booking.UserID,
			)
		}
		return model.SeatBooked, fmt.Errorf("seat %s is already held or confirmed", booking.SeatID)
	}

	// Copy the stored session details back to the caller (session_id, expires_at, etc.).
	*booking = session

	if bs.logger != nil {
		bs.logger.Info(
			"Seat held",
			"seat_id", booking.SeatID,
			"movie_id", booking.MovieID,
			"user_id", booking.UserID,
			"status", session.Status,
		)
	}

	return session.Status, nil
}

func (bs *BookingService) GetBookedSeats(ctx context.Context, movieID string) ([]model.Booking, error) {
	if movieID == "" {
		return nil, fmt.Errorf("movie_id is required")
	}
	if bs.rdb == nil {
		return nil, fmt.Errorf("redis store is nil")
	}

	out := bs.rdb.GetBookedSeats(movieID)
	return out, nil
}

func (bs *BookingService) Confirm(ctx context.Context, sessionID string, userID string) (model.Booking, error) {
	if bs.rdb == nil {
		return model.Booking{}, fmt.Errorf("redis store is nil")
	}
	return bs.rdb.Confirm(ctx, sessionID, userID)
}

func (bs *BookingService) Release(ctx context.Context, sessionID string, userID string) error {
	if bs.rdb == nil {
		return fmt.Errorf("redis store is nil")
	}
	return bs.rdb.Release(ctx, sessionID, userID)
}
