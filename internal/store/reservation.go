package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"stockrush-go/internal/domain"
)

type ReservationRequest struct {
	UserID         string `json:"userId"`
	Quantity       int    `json:"quantity"`
	IdempotencyKey string `json:"-"`
}

type ReservationResult struct {
	Reservation domain.Reservation `json:"reservation"`
	Duplicate   bool               `json:"duplicate"`
}

type idempotencyResponse struct {
	ErrorCode string             `json:"errorCode,omitempty"`
	Message   string             `json:"message,omitempty"`
	Result    *ReservationResult `json:"result,omitempty"`
}

func (s *Store) Reserve(ctx context.Context, saleID string, request ReservationRequest) (ReservationResult, error) {
	request.UserID = strings.TrimSpace(request.UserID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if saleID == "" || request.UserID == "" || len(request.UserID) > 128 || request.Quantity <= 0 || request.Quantity > 100 || request.IdempotencyKey == "" || len(request.IdempotencyKey) > 200 {
		return ReservationResult{}, &domain.Error{Code: "VALIDATION_ERROR", Message: "sale, user, quantity, and Idempotency-Key are required", HTTPStatus: 400}
	}
	fingerprint := fingerprint(saleID, request.UserID, request.Quantity)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ReservationResult{}, err
	}
	defer rollback(tx)

	var inserted string
	err = tx.QueryRow(ctx, `INSERT INTO idempotency_keys(key,scope,user_id,request_fingerprint,expires_at)
		VALUES ($1,'reservation',$2,$3,now()+interval '24 hours') ON CONFLICT DO NOTHING RETURNING key`,
		request.IdempotencyKey, request.UserID, fingerprint).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.replayReservation(ctx, tx, request.IdempotencyKey, request.UserID, fingerprint)
	}
	if err != nil {
		return ReservationResult{}, fmt.Errorf("claim idempotency key: %w", err)
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, saleID+":"+request.UserID); err != nil {
		return ReservationResult{}, err
	}
	var productID, saleState, currency string
	var startsAt, endsAt time.Time
	var maxQuantity int
	var priceMinor int64
	var productActive bool
	err = tx.QueryRow(ctx, `SELECT s.product_id,s.state,s.starts_at,s.ends_at,s.max_quantity_per_user,p.price_minor,p.currency,p.active
		FROM flash_sales s JOIN products p ON p.id=s.product_id WHERE s.id=$1`, saleID).
		Scan(&productID, &saleState, &startsAt, &endsAt, &maxQuantity, &priceMinor, &currency, &productActive)
	if err != nil {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, domainError(notFound(err), "SALE_NOT_FOUND", "Flash sale was not found", 404))
	}
	now := s.now().UTC()
	if saleState != string(domain.SaleActive) || now.Before(startsAt) || !now.Before(endsAt) {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, &domain.Error{Code: "SALE_NOT_ACTIVE", Message: "The flash sale is not active", HTTPStatus: 409})
	}
	if !productActive {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, &domain.Error{Code: "PRODUCT_INACTIVE", Message: "The product is inactive", HTTPStatus: 409})
	}
	var existingQuantity int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(quantity),0) FROM reservations
		WHERE sale_id=$1 AND user_id=$2 AND state IN ('pending','confirmed','paid')`, saleID, request.UserID).Scan(&existingQuantity); err != nil {
		return ReservationResult{}, err
	}
	if existingQuantity+request.Quantity > maxQuantity {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, &domain.Error{Code: "PURCHASE_LIMIT_EXCEEDED", Message: "The per-user purchase limit would be exceeded", HTTPStatus: 409})
	}
	saleResult, err := tx.Exec(ctx, `UPDATE flash_sales SET remaining_stock=remaining_stock-$2,updated_at=now()
		WHERE id=$1 AND state='active' AND remaining_stock >= $2`, saleID, request.Quantity)
	if err != nil {
		return ReservationResult{}, err
	}
	if saleResult.RowsAffected() == 0 {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, &domain.Error{Code: "INVENTORY_SOLD_OUT", Message: "The flash sale allocation is sold out", HTTPStatus: 409})
	}
	result, err := tx.Exec(ctx, `UPDATE inventory SET available=available-$1,reserved=reserved+$1,updated_at=now()
		WHERE product_id=$2 AND available >= $1`, request.Quantity, productID)
	if err != nil {
		return ReservationResult{}, err
	}
	if result.RowsAffected() == 0 {
		return s.finishReservationError(ctx, tx, request.IdempotencyKey, &domain.Error{Code: "INVENTORY_SOLD_OUT", Message: "The requested product is sold out", HTTPStatus: 409})
	}

	reservationID, err := domain.NewID()
	if err != nil {
		return ReservationResult{}, err
	}
	orderID, err := domain.NewID()
	if err != nil {
		return ReservationResult{}, err
	}
	expiresAt := now.Add(s.ttl)
	var reservation domain.Reservation
	err = tx.QueryRow(ctx, `INSERT INTO reservations(id,sale_id,product_id,user_id,quantity,state,expires_at)
		VALUES ($1,$2,$3,$4,$5,'pending',$6)
		RETURNING id,sale_id,product_id,user_id,quantity,state,expires_at,created_at,updated_at`,
		reservationID, saleID, productID, request.UserID, request.Quantity, expiresAt).
		Scan(&reservation.ID, &reservation.SaleID, &reservation.ProductID, &reservation.UserID, &reservation.Quantity, &reservation.State, &reservation.ExpiresAt, &reservation.CreatedAt, &reservation.UpdatedAt)
	if err != nil {
		return ReservationResult{}, fmt.Errorf("insert reservation: %w", err)
	}
	reservation.OrderID = orderID
	if _, err := tx.Exec(ctx, `INSERT INTO orders(id,reservation_id,sale_id,product_id,user_id,quantity,amount_minor,currency,state)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending')`, orderID, reservationID, saleID, productID, request.UserID, request.Quantity, priceMinor*int64(request.Quantity), currency); err != nil {
		return ReservationResult{}, fmt.Errorf("insert order: %w", err)
	}
	if err := event(ctx, tx, "reservation", reservationID, "reservation.created", "", "pending", map[string]any{"quantity": request.Quantity}); err != nil {
		return ReservationResult{}, err
	}
	response := idempotencyResponse{Result: &ReservationResult{Reservation: reservation}}
	body, _ := json.Marshal(response)
	if _, err := tx.Exec(ctx, `UPDATE idempotency_keys SET resource_id=$2,response_status=201,response_body=$3 WHERE key=$1`, request.IdempotencyKey, reservationID, body); err != nil {
		return ReservationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReservationResult{}, err
	}
	return *response.Result, nil
}

func (s *Store) replayReservation(ctx context.Context, tx pgx.Tx, key, userID, fingerprint string) (ReservationResult, error) {
	var storedUser, storedFingerprint string
	var status *int
	var body []byte
	err := tx.QueryRow(ctx, `SELECT user_id,request_fingerprint,response_status,response_body FROM idempotency_keys WHERE key=$1 AND scope='reservation'`, key).
		Scan(&storedUser, &storedFingerprint, &status, &body)
	if err != nil {
		return ReservationResult{}, err
	}
	if storedUser != userID || storedFingerprint != fingerprint {
		return ReservationResult{}, &domain.Error{Code: "IDEMPOTENCY_CONFLICT", Message: "Idempotency-Key was already used with a different request", HTTPStatus: 409}
	}
	if status == nil || len(body) == 0 {
		return ReservationResult{}, &domain.Error{Code: "IDEMPOTENCY_IN_PROGRESS", Message: "The original request is still processing", HTTPStatus: 409}
	}
	var response idempotencyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ReservationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReservationResult{}, err
	}
	if response.ErrorCode != "" {
		return ReservationResult{}, &domain.Error{Code: response.ErrorCode, Message: response.Message, HTTPStatus: *status}
	}
	response.Result.Duplicate = true
	return *response.Result, nil
}

func (s *Store) finishReservationError(ctx context.Context, tx pgx.Tx, key string, domainErr *domain.Error) (ReservationResult, error) {
	body, _ := json.Marshal(idempotencyResponse{ErrorCode: domainErr.Code, Message: domainErr.Message})
	if _, err := tx.Exec(ctx, `UPDATE idempotency_keys SET response_status=$2,response_body=$3 WHERE key=$1`, key, domainErr.HTTPStatus, body); err != nil {
		return ReservationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReservationResult{}, err
	}
	return ReservationResult{}, domainErr
}

func fingerprint(saleID, userID string, quantity int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", saleID, userID, quantity)))
	return hex.EncodeToString(sum[:])
}

func domainError(err error, code, message string, status int) *domain.Error {
	var de *domain.Error
	if errors.As(err, &de) {
		return &domain.Error{Code: code, Message: message, HTTPStatus: status}
	}
	return &domain.Error{Code: "INTERNAL_ERROR", Message: "An internal error occurred", HTTPStatus: http.StatusInternalServerError}
}

func (s *Store) GetReservation(ctx context.Context, id string) (domain.Reservation, error) {
	var reservation domain.Reservation
	err := s.pool.QueryRow(ctx, `SELECT r.id,r.sale_id,r.product_id,o.id,r.user_id,r.quantity,r.state,r.expires_at,r.created_at,r.updated_at
		FROM reservations r JOIN orders o ON o.reservation_id=r.id WHERE r.id=$1`, id).
		Scan(&reservation.ID, &reservation.SaleID, &reservation.ProductID, &reservation.OrderID, &reservation.UserID, &reservation.Quantity, &reservation.State, &reservation.ExpiresAt, &reservation.CreatedAt, &reservation.UpdatedAt)
	return reservation, notFound(err)
}

func (s *Store) TransitionReservation(ctx context.Context, id string, target domain.ReservationState) (domain.Reservation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Reservation{}, err
	}
	defer rollback(tx)
	var current domain.ReservationState
	var productID, saleID string
	var quantity int
	err = tx.QueryRow(ctx, `SELECT state,product_id,sale_id,quantity FROM reservations WHERE id=$1 FOR UPDATE`, id).Scan(&current, &productID, &saleID, &quantity)
	if err != nil {
		return domain.Reservation{}, notFound(err)
	}
	if err := domain.ValidateReservationTransition(current, target); err != nil {
		return domain.Reservation{}, &domain.Error{Code: "INVALID_STATE_TRANSITION", Message: err.Error(), HTTPStatus: 409}
	}
	if _, err := tx.Exec(ctx, `UPDATE reservations SET state=$2,updated_at=now() WHERE id=$1`, id, target); err != nil {
		return domain.Reservation{}, err
	}
	orderState := string(target)
	if _, err := tx.Exec(ctx, `UPDATE orders SET state=$2,updated_at=now() WHERE reservation_id=$1`, id, orderState); err != nil {
		return domain.Reservation{}, err
	}
	if target == domain.ReservationCancelled {
		if _, err := tx.Exec(ctx, `UPDATE inventory SET available=available+$2,reserved=reserved-$2,updated_at=now() WHERE product_id=$1 AND reserved >= $2`, productID, quantity); err != nil {
			return domain.Reservation{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE flash_sales SET remaining_stock=remaining_stock+$2,updated_at=now() WHERE id=$1`, saleID, quantity); err != nil {
			return domain.Reservation{}, err
		}
	}
	if err := event(ctx, tx, "reservation", id, "reservation."+string(target), string(current), string(target), nil); err != nil {
		return domain.Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Reservation{}, err
	}
	return s.GetReservation(ctx, id)
}
