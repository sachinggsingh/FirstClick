package model

import (
	"context"
	"time"
)

type SeatStatus string

const (
	SeatAvailable SeatStatus = "available"
	SeatBooked    SeatStatus = "booked"
	SeatReserved  SeatStatus = "reserved"
	SeatHold      SeatStatus = "hold"
	SeatConfirm   SeatStatus = "confirmed"
)

type Booking struct {
	ID        string
	UserID    string
	MovieID   string
	SeatID    string
	Status    SeatStatus
	ExpiresAt time.Time
}

type BookingSeat interface {
	Book(ctx context.Context, booking *Booking) (SeatStatus, error)
	GetBookedSeats(ctx context.Context, movieID string) ([]Booking, error)
}
