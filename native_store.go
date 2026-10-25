package main

import (
	"context"
	"encoding/json"
	"openpak/np-trophy/native"
)

// nativeStore binds the console protocol to the existing database without an HTTP hop.
type nativeStore struct{ store Trophies }

func (s nativeStore) Games(ctx context.Context, account string) ([]native.TrophyGame, error) {
	return s.store.Games(ctx, account)
}
func (s nativeStore) Exchange(ctx context.Context, account string, payload []byte) ([]native.TrophyGame, error) {
	if payload != nil {
		var in struct {
			CommunicationID string   `json:"communication_id"`
			Trophies        []Unlock `json:"trophies"`
		}
		if err := json.Unmarshal(payload, &in); err != nil {
			return nil, err
		}
		if err := s.store.Merge(ctx, account, in.CommunicationID, in.Trophies); err != nil {
			return nil, err
		}
	}
	return s.Games(ctx, account)
}
