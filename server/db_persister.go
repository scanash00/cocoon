package server

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/models"
	"github.com/haileyok/cocoon/internal/db"
	dbmodels "github.com/haileyok/cocoon/models"
)

type DbPersister struct {
	db        *db.DB
	lk        sync.Mutex
	seq       int64
	broadcast func(*events.XRPCStreamEvent)
}

func NewDbPersister(database *db.DB) *DbPersister {
	var seqRecord dbmodels.EventSequence
	if err := database.Raw(context.Background(), "SELECT seq FROM event_sequences WHERE id = 1", nil).Scan(&seqRecord).Error; err != nil {
	}

	return &DbPersister{
		db:  database,
		seq: seqRecord.Seq,
	}
}

func (dp *DbPersister) Persist(ctx context.Context, e *events.XRPCStreamEvent) error {
	dp.lk.Lock()
	defer dp.lk.Unlock()

	dp.seq++
	seq := dp.seq

	switch {
	case e.RepoCommit != nil:
		e.RepoCommit.Seq = seq
	case e.RepoSync != nil:
		e.RepoSync.Seq = seq
	case e.RepoIdentity != nil:
		e.RepoIdentity.Seq = seq
	case e.RepoAccount != nil:
		e.RepoAccount.Seq = seq
	case e.LabelLabels != nil:
		e.LabelLabels.Seq = seq
	default:
		return nil
	}

	buf := new(bytes.Buffer)
	if err := e.Serialize(buf); err != nil {
		return err
	}

	now := time.Now()
	if err := dp.db.Exec(ctx, "INSERT INTO events (seq, data, created_at) VALUES (?, ?, ?)",
		nil, seq, buf.Bytes(), now).Error; err != nil {
		return err
	}

	dp.db.Exec(ctx, "INSERT INTO event_sequences (id, seq) VALUES (1, ?) ON CONFLICT (id) DO UPDATE SET seq = ?",
		nil, seq, seq)

	if dp.broadcast != nil {
		dp.broadcast(e)
	}

	return nil
}

func (dp *DbPersister) Playback(ctx context.Context, since int64, cb func(*events.XRPCStreamEvent) error) error {
	rows, err := dp.db.Raw(ctx, "SELECT data FROM events WHERE seq > ? ORDER BY seq ASC", nil, since).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}

		evt := &events.XRPCStreamEvent{}
		if err := evt.Deserialize(bytes.NewReader(data)); err != nil {
			continue
		}

		if err := cb(evt); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (dp *DbPersister) TakeDownRepo(ctx context.Context, usr models.Uid) error {
	return nil
}

func (dp *DbPersister) Flush(ctx context.Context) error {
	return nil
}

func (dp *DbPersister) Shutdown(ctx context.Context) error {
	return nil
}

func (dp *DbPersister) SetEventBroadcaster(bc func(*events.XRPCStreamEvent)) {
	dp.broadcast = bc
}

func (dp *DbPersister) CleanupOldEvents(ctx context.Context, retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return dp.db.Exec(ctx, "DELETE FROM events WHERE created_at < ?", nil, cutoff).Error
}
func (dp *DbPersister) GetOldestSeq(ctx context.Context) (int64, error) {
	var result struct {
		Seq int64
	}
	if err := dp.db.Raw(ctx, "SELECT MIN(seq) as seq FROM events", nil).Scan(&result).Error; err != nil {
		return 0, err
	}
	return result.Seq, nil
}
