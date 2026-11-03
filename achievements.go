package main

import (
	"context"
	"sort"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "openpak/np-trophy/internal/achievementspb"
	"openpak/np-trophy/native"
)

// network is the account-core link namespace the achievements service files PS3 trophies under.
const network = "np"

// Unlock is one earned trophy. IDs are the firmware's zero-based trophy index, EarnedAt is
// Unix seconds and Grade is "", "B", "S", "G" or "P".
type Unlock = native.Unlock

// Game is the native trophy set of one communication id.
type Game = native.TrophyGame

// Achievements is np-trophy's storage: the neutral achievements service. The service enforces
// the rules this server always had: the first recorded unlock of a trophy is kept (a later sync
// never moves its time) and a grade only fills in where none was recorded yet.
type Achievements struct{ Client pb.AchievementsClient }

// DialAchievements connects to the achievements service (gRPC) and presents the internal key on
// every call. extra lets tests add a dialer.
func DialAchievements(addr, key string, extra ...grpc.DialOption) (*Achievements, *grpc.ClientConn, error) {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, inv grpc.UnaryInvoker, o ...grpc.CallOption) error {
			return inv(metadata.AppendToOutgoingContext(ctx, "x-internal-key", key), method, req, reply, cc, o...)
		}),
	}, extra...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, nil, err
	}
	return &Achievements{Client: pb.NewAchievementsClient(conn)}, conn, nil
}

// Merge records unlocks of one communication id for an account.
func (a *Achievements) Merge(ctx context.Context, account, communicationID string, unlocks []Unlock) error {
	if len(unlocks) == 0 {
		return nil
	}
	items := make([]*pb.UnlockItem, 0, len(unlocks))
	for _, u := range unlocks {
		items = append(items, &pb.UnlockItem{AchievementId: strconv.Itoa(u.ID), UnlockedUnix: u.EarnedAt, Grade: u.Grade})
	}
	_, err := a.Client.Unlock(ctx, &pb.UnlockRequest{Ref: &pb.TitleRef{Network: network, Title: communicationID}, Account: account, Unlocks: items})
	return err
}

// Games returns every trophy set the account has unlocks in, by communication ID and trophy ID.
// The service answers with every title mapped on the "np" network, so a PS4 or PS5 set of a
// game the account also earned elsewhere shows under its PS3 communication id too.
func (a *Achievements) Games(ctx context.Context, account string) ([]Game, error) {
	res, err := a.Client.ListUnlocks(ctx, &pb.ListUnlocksRequest{Ref: &pb.TitleRef{Network: network}, Account: account})
	if err != nil {
		return nil, err
	}
	games := []Game{}
	for _, t := range res.GetTitles() {
		if t.GetNativeId() == "" {
			continue
		}
		g := Game{CommunicationID: t.GetNativeId()}
		for _, u := range t.GetUnlocks() {
			id, err := strconv.Atoi(u.GetAchievementId())
			if err != nil || u.GetUnlockedUnix() == 0 {
				continue
			}
			g.Trophies = append(g.Trophies, Unlock{ID: id, EarnedAt: u.GetUnlockedUnix(), Grade: letter(u.GetGrade())})
		}
		if len(g.Trophies) == 0 {
			continue
		}
		sort.Slice(g.Trophies, func(i, j int) bool { return g.Trophies[i].ID < g.Trophies[j].ID })
		games = append(games, g)
	}
	sort.SliceStable(games, func(i, j int) bool { return games[i].CommunicationID < games[j].CommunicationID })
	return games, nil
}

func letter(g string) string {
	switch g {
	case "bronze":
		return "B"
	case "silver":
		return "S"
	case "gold":
		return "G"
	case "platinum":
		return "P"
	}
	return ""
}
