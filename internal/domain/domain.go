package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type ReservationState string

const (
	ReservationPending   ReservationState = "pending"
	ReservationConfirmed ReservationState = "confirmed"
	ReservationPaid      ReservationState = "paid"
	ReservationCancelled ReservationState = "cancelled"
	ReservationExpired   ReservationState = "expired"
)

type SaleState string

const (
	SaleDraft     SaleState = "draft"
	SaleScheduled SaleState = "scheduled"
	SaleActive    SaleState = "active"
	SaleEnded     SaleState = "ended"
	SaleCancelled SaleState = "cancelled"
)

type Product struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceMinor  int64     `json:"priceMinor"`
	Currency    string    `json:"currency"`
	Active      bool      `json:"active"`
	Available   int       `json:"available"`
	Reserved    int       `json:"reserved"`
	Sold        int       `json:"sold"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Sale struct {
	ID                 string    `json:"id"`
	ProductID          string    `json:"productId"`
	StartsAt           time.Time `json:"startsAt"`
	EndsAt             time.Time `json:"endsAt"`
	AllocatedStock     int       `json:"allocatedStock"`
	RemainingStock     int       `json:"remainingStock"`
	MaxQuantityPerUser int       `json:"maxQuantityPerUser"`
	State              SaleState `json:"state"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Reservation struct {
	ID        string           `json:"id"`
	SaleID    string           `json:"saleId"`
	ProductID string           `json:"productId"`
	OrderID   string           `json:"orderId"`
	UserID    string           `json:"userId"`
	Quantity  int              `json:"quantity"`
	State     ReservationState `json:"state"`
	ExpiresAt time.Time        `json:"expiresAt"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

type Order struct {
	ID            string    `json:"id"`
	ReservationID string    `json:"reservationId"`
	SaleID        string    `json:"saleId"`
	ProductID     string    `json:"productId"`
	UserID        string    `json:"userId"`
	Quantity      int       `json:"quantity"`
	AmountMinor   int64     `json:"amountMinor"`
	Currency      string    `json:"currency"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Payment struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"orderId"`
	IdempotencyKey string    `json:"idempotencyKey"`
	Outcome        string    `json:"outcome"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string { return e.Message }

func IsCode(err error, code string) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

func ValidateReservationTransition(from, to ReservationState) error {
	valid := (from == ReservationPending && (to == ReservationConfirmed || to == ReservationCancelled || to == ReservationExpired)) ||
		(from == ReservationConfirmed && to == ReservationPaid)
	if !valid {
		return fmt.Errorf("invalid reservation transition %s -> %s", from, to)
	}
	return nil
}

func NewID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" + hex.EncodeToString(b[4:6]) + "-" + hex.EncodeToString(b[6:8]) + "-" + hex.EncodeToString(b[8:10]) + "-" + hex.EncodeToString(b[10:16]), nil
}
