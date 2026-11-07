package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// TranscodeQueueKey is the Redis list for pending transcode jobs.
const TranscodeQueueKey = "transcode:queue"

// TranscodePayload is the worker job envelope.
type TranscodePayload struct {
	JobID        string `json:"jobId"`
	TrackID      string `json:"trackId"`
	InputBucket  string `json:"inputBucket"`
	InputKey     string `json:"inputKey"`
	OutputPrefix string `json:"outputPrefix"`
}

// Queue wraps the Redis transcode queue.
type Queue struct{ rdb *redis.Client }

// EnqueueTranscode pushes a job onto the tail (FIFO with DequeueTranscode).
func (q *Queue) EnqueueTranscode(ctx context.Context, p TranscodePayload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return q.rdb.RPush(ctx, TranscodeQueueKey, string(b)).Err()
}

// DequeueTranscode blocks up to timeout for the next job.
// It returns (nil, nil) on timeout.
func (q *Queue) DequeueTranscode(ctx context.Context, timeout time.Duration) (*TranscodePayload, error) {
	res, err := q.rdb.BLPop(ctx, timeout, TranscodeQueueKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if len(res) < 2 {
		return nil, nil
	}
	var p TranscodePayload
	if err := json.Unmarshal([]byte(res[1]), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// QueueDepth returns the pending job count.
func (q *Queue) QueueDepth(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, TranscodeQueueKey).Result()
}

// Repos aggregates all repositories over one pool and one Redis client.
type Repos struct {
	DB            *sql.DB
	RDB           *redis.Client
	Users         *UserRepo
	Partners      *PartnerRepo
	Subscriptions *SubscriptionRepo
	Policies      *PolicyRepo
	Legal         *LegalRepo
	Albums        *AlbumRepo
	Tracks        *TrackRepo
	Rights        *RightsRepo
	Jobs          *JobRepo
	Disputes      *DisputeRepo
	Playlists     *PlaylistRepo
	Playback      *PlaybackRepo
	Streaming     *StreamingRepo
	Settlements   *SettlementRepo
	Blocks        *BlockRepo
	Gate          *GateStore
	Queue         *Queue
	Resets        *PasswordResetRepo
	OAuth         *OAuthRepo
	Billing       *BillingRepo
}

// WithTx returns a repository aggregate whose catalog writes (albums,
// tracks, rights, jobs) run inside tx. Reads through other repositories
// keep using the pool. Callers commit or roll back the transaction.
func (r *Repos) WithTx(tx *sql.Tx) *Repos {
	clone := *r
	clone.Albums = &AlbumRepo{db: tx}
	clone.Tracks = &TrackRepo{db: tx}
	clone.Rights = &RightsRepo{db: tx}
	clone.Jobs = &JobRepo{db: tx}
	return &clone
}

// New builds the repository aggregate.
func New(db *sql.DB, rdb *redis.Client) *Repos {
	return &Repos{
		DB:            db,
		RDB:           rdb,
		Users:         &UserRepo{db: db},
		Partners:      &PartnerRepo{db: db},
		Subscriptions: &SubscriptionRepo{db: db},
		Policies:      &PolicyRepo{db: db},
		Legal:         &LegalRepo{db: db},
		Albums:        &AlbumRepo{db: db},
		Tracks:        &TrackRepo{db: db},
		Rights:        &RightsRepo{db: db},
		Jobs:          &JobRepo{db: db},
		Disputes:      &DisputeRepo{db: db},
		Playlists:     &PlaylistRepo{db: db},
		Playback:      &PlaybackRepo{db: db},
		Streaming:     &StreamingRepo{db: db},
		Settlements:   &SettlementRepo{db: db},
		Blocks:        &BlockRepo{db: db},
		Gate:          &GateStore{db: db},
		Queue:         &Queue{rdb: rdb},
		Resets:        &PasswordResetRepo{db: db},
		OAuth:         &OAuthRepo{db: db},
		Billing:       &BillingRepo{db: db},
	}
}
