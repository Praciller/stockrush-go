package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"stockrush-go/internal/domain"
)

type PaymentRequest struct {
	OrderID        string `json:"orderId"`
	Outcome        string `json:"outcome"`
	IdempotencyKey string `json:"-"`
}

type PaymentResult struct {
	Payment   domain.Payment `json:"payment"`
	Duplicate bool           `json:"duplicate"`
}

func (s *Store) SimulatePayment(ctx context.Context, request PaymentRequest) (PaymentResult, error) {
	request.OrderID = strings.TrimSpace(request.OrderID)
	request.Outcome = strings.ToLower(strings.TrimSpace(request.Outcome))
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if request.OrderID == "" || request.IdempotencyKey == "" || (request.Outcome != "successful" && request.Outcome != "failed" && request.Outcome != "delayed") {
		return PaymentResult{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "orderId, valid outcome, and Idempotency-Key are required", HTTPStatus: 400}
	}
	fingerprintBytes := sha256.Sum256([]byte(request.OrderID + "\x00" + request.Outcome))
	fingerprint := hex.EncodeToString(fingerprintBytes[:])
	paymentID, err := domain.NewID()
	if err != nil {
		return PaymentResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PaymentResult{}, err
	}
	defer rollback(tx)
	var inserted string
	err = tx.QueryRow(ctx, `INSERT INTO payments(id,order_id,idempotency_key,outcome,status,request_fingerprint)
		VALUES ($1,$2,$3,$4,'pending',$5) ON CONFLICT (idempotency_key) DO NOTHING RETURNING id`,
		paymentID, request.OrderID, request.IdempotencyKey, request.Outcome, fingerprint).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		var payment domain.Payment
		var storedFingerprint string
		err = tx.QueryRow(ctx, `SELECT id,order_id,idempotency_key,outcome,status,request_fingerprint,created_at,updated_at FROM payments WHERE idempotency_key=$1`, request.IdempotencyKey).
			Scan(&payment.ID, &payment.OrderID, &payment.IdempotencyKey, &payment.Outcome, &payment.Status, &storedFingerprint, &payment.CreatedAt, &payment.UpdatedAt)
		if err != nil {
			return PaymentResult{}, err
		}
		if storedFingerprint != fingerprint {
			return PaymentResult{}, &domain.Error{Code: "IDEMPOTENCY_CONFLICT", Message: "Idempotency-Key was already used with a different payment request", HTTPStatus: 409}
		}
		if err := tx.Commit(ctx); err != nil {
			return PaymentResult{}, err
		}
		return PaymentResult{Payment: payment, Duplicate: true}, nil
	}
	if err != nil {
		return PaymentResult{}, err
	}

	var reservationID, reservationState, productID string
	var quantity int
	err = tx.QueryRow(ctx, `SELECT o.reservation_id,r.state,r.product_id,r.quantity
		FROM orders o JOIN reservations r ON r.id=o.reservation_id WHERE o.id=$1 FOR UPDATE OF o,r`, request.OrderID).
		Scan(&reservationID, &reservationState, &productID, &quantity)
	if err != nil {
		return PaymentResult{}, notFound(err)
	}
	status := "pending"
	switch request.Outcome {
	case "successful":
		if reservationState != string(domain.ReservationConfirmed) {
			return PaymentResult{}, &domain.Error{Code: "INVALID_STATE_TRANSITION", Message: "Only confirmed reservations can be paid", HTTPStatus: 409}
		}
		result, err := tx.Exec(ctx, `UPDATE inventory SET reserved=reserved-$2,sold=sold+$2,updated_at=now() WHERE product_id=$1 AND reserved >= $2`, productID, quantity)
		if err != nil {
			return PaymentResult{}, fmt.Errorf("reconcile paid inventory: %w", err)
		}
		if result.RowsAffected() != 1 {
			return PaymentResult{}, errors.New("paid inventory invariant prevented reconciliation")
		}
		if _, err := tx.Exec(ctx, `UPDATE reservations SET state='paid',updated_at=now() WHERE id=$1`, reservationID); err != nil {
			return PaymentResult{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET state='paid',updated_at=now() WHERE id=$1`, request.OrderID); err != nil {
			return PaymentResult{}, err
		}
		if err := event(ctx, tx, "reservation", reservationID, "reservation.paid", "confirmed", "paid", nil); err != nil {
			return PaymentResult{}, err
		}
		status = "succeeded"
	case "failed":
		if reservationState != string(domain.ReservationConfirmed) && reservationState != string(domain.ReservationPending) {
			return PaymentResult{}, &domain.Error{Code: "INVALID_STATE_TRANSITION", Message: "Payment cannot fail for a completed reservation", HTTPStatus: 409}
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET state='payment_failed',updated_at=now() WHERE id=$1`, request.OrderID); err != nil {
			return PaymentResult{}, err
		}
		status = "failed"
	}
	var payment domain.Payment
	err = tx.QueryRow(ctx, `UPDATE payments SET status=$2,updated_at=now() WHERE id=$1
		RETURNING id,order_id,idempotency_key,outcome,status,created_at,updated_at`, paymentID, status).
		Scan(&payment.ID, &payment.OrderID, &payment.IdempotencyKey, &payment.Outcome, &payment.Status, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return PaymentResult{}, err
	}
	if err := event(ctx, tx, "payment", paymentID, "payment."+status, "", status, map[string]string{"outcome": request.Outcome}); err != nil {
		return PaymentResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PaymentResult{}, err
	}
	return PaymentResult{Payment: payment}, nil
}
