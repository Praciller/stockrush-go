package domain

import "testing"

func TestReservationStateTransitions(t *testing.T) {
	valid := map[ReservationState][]ReservationState{
		ReservationPending:   {ReservationConfirmed, ReservationCancelled, ReservationExpired},
		ReservationConfirmed: {ReservationPaid},
	}
	for from, targets := range valid {
		for _, to := range targets {
			if err := ValidateReservationTransition(from, to); err != nil {
				t.Fatalf("transition %s -> %s error = %v", from, to, err)
			}
		}
	}
	for _, transition := range [][2]ReservationState{
		{ReservationExpired, ReservationPaid},
		{ReservationCancelled, ReservationPaid},
		{ReservationPaid, ReservationExpired},
		{ReservationConfirmed, ReservationExpired},
	} {
		if err := ValidateReservationTransition(transition[0], transition[1]); err == nil {
			t.Fatalf("transition %s -> %s error = nil", transition[0], transition[1])
		}
	}
}
