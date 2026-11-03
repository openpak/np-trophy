// Package fakeachievements is an in-process openpak.achievements.v1.Achievements gRPC server for
// tests: unlocks live in memory, the first recorded time and the fill-once grade follow the real
// service, titles map by (network, native id) with PlayStation communication ids registering
// themselves, and the x-internal-key check is mirrored.
package fakeachievements

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "openpak/np-trophy/internal/achievementspb"
)

type row struct {
	at       int64
	grade    string
	progress uint32
}

type Fake struct {
	pb.UnimplementedAchievementsServer
	Key string
	// Down makes every call fail Unavailable.
	Down  bool
	mu    sync.Mutex
	maps  map[string]string          // network|native -> title
	rows  map[string]map[string]*row // account|title -> achievement -> row
	Calls int                        // Unlock calls
}

func New(apiKey string) *Fake {
	return &Fake{Key: apiKey, maps: map[string]string{}, rows: map[string]map[string]*row{}}
}

// Map joins a native id to a title key, like the real MapTitle.
func (f *Fake) Map(network, native, title string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.maps[network+"|"+native] = title
}

// Serve starts the fake and returns dial options for a client.
func (f *Fake) Serve(t testing.TB) []grpc.DialOption {
	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer()
	pb.RegisterAchievementsServer(gs, f)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)
	return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() })}
}

func (f *Fake) check(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get("x-internal-key"); len(v) == 0 || v[0] != f.Key {
		return status.Error(codes.Unauthenticated, "missing internal credentials")
	}
	if f.Down {
		return status.Error(codes.Unavailable, "down")
	}
	return nil
}

func norm(g string) string {
	switch strings.ToUpper(g) {
	case "B", "BRONZE":
		return "bronze"
	case "S", "SILVER":
		return "silver"
	case "G", "GOLD":
		return "gold"
	case "P", "PLATINUM":
		return "platinum"
	}
	return ""
}

func (f *Fake) title(ref *pb.TitleRef) string {
	k := ref.GetNetwork() + "|" + ref.GetTitle()
	if t, ok := f.maps[k]; ok {
		return t
	}
	t := ref.GetNetwork() + ":" + ref.GetTitle()
	f.maps[k] = t
	return t
}

func (f *Fake) Unlock(ctx context.Context, req *pb.UnlockRequest) (*pb.UnlockResponse, error) {
	if err := f.check(ctx); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	title := f.title(req.GetRef())
	k := req.GetAccount() + "|" + title
	if f.rows[k] == nil {
		f.rows[k] = map[string]*row{}
	}
	out := &pb.UnlockResponse{Title: title}
	for _, u := range req.GetUnlocks() {
		r := f.rows[k][u.GetAchievementId()]
		newly := false
		if r == nil {
			r = &row{}
			f.rows[k][u.GetAchievementId()] = r
		}
		if r.at == 0 && u.GetUnlockedUnix() > 0 {
			r.at, newly = u.GetUnlockedUnix(), true
		}
		if r.grade == "" {
			r.grade = norm(u.GetGrade())
		}
		out.Results = append(out.Results, &pb.Unlocked{AchievementId: u.GetAchievementId(), UnlockedUnix: r.at, Grade: r.grade, NewlyUnlocked: newly})
	}
	return out, nil
}

func (f *Fake) ListUnlocks(ctx context.Context, req *pb.ListUnlocksRequest) (*pb.ListUnlocksResponse, error) {
	if err := f.check(ctx); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	net := req.GetRef().GetNetwork()
	native := map[string]string{} // title -> lowest native id on the network
	for k, t := range f.maps {
		n, id, _ := strings.Cut(k, "|")
		if n == net && (native[t] == "" || id < native[t]) {
			native[t] = id
		}
	}
	out := &pb.ListUnlocksResponse{}
	var keys []string
	for k := range f.rows {
		if strings.HasPrefix(k, req.GetAccount()+"|") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		title := strings.TrimPrefix(k, req.GetAccount()+"|")
		if native[title] == "" {
			continue
		}
		tu := &pb.TitleUnlocks{Title: title, NativeId: native[title]}
		var ids []string
		for id := range f.rows[k] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			r := f.rows[k][id]
			if r.at == 0 && !req.GetIncludeProgress() {
				continue
			}
			tu.Unlocks = append(tu.Unlocks, &pb.Unlocked{AchievementId: id, UnlockedUnix: r.at, Grade: r.grade, Network: net})
		}
		out.Titles = append(out.Titles, tu)
	}
	return out, nil
}
