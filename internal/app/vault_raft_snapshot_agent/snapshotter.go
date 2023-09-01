package vault_raft_snapshot_agent

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/upload"
	"github.com/Argelbargel/vault-raft-snapshot-agent/internal/app/vault_raft_snapshot_agent/vault"
	"go.uber.org/multierr"
)

type SnapshotterConfig struct {
	Vault     vault.VaultClientConfig
	Snapshots SnapshotConfig
	Uploaders upload.UploadersConfig
}

type SnapshotConfig struct {
	Frequency       time.Duration `default:"1h"`
	Retain          int
	Timeout         time.Duration `default:"60s"`
	NamePrefix      string        `default:"raft-snapshot-"`
	NameSuffix      string        `default:".snap"`
	TimestampFormat string        `default:"2006-01-02T15-04-05Z-0700"`
}

type Snapshotter struct {
	lock      sync.Mutex
	client    *vault.VaultClient
	uploaders []upload.Uploader
	config    SnapshotConfig
}

func CreateSnapshotter(config SnapshotterConfig) (*Snapshotter, error) {
	snapshotter := &Snapshotter{}

	err := snapshotter.Reconfigure(config)
	return snapshotter, err
}

func (s *Snapshotter) Reconfigure(config SnapshotterConfig) error {
	client, err := vault.CreateClient(config.Vault)
	if err != nil {
		return err
	}

	uploaders, err := upload.CreateUploaders(config.Uploaders)
	if err != nil {
		return err
	}

	s.Configure(config.Snapshots, client, uploaders)
	return nil
}

func (s *Snapshotter) Configure(config SnapshotConfig, client *vault.VaultClient, uploaders []upload.Uploader) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.client = client
	s.uploaders = uploaders
	s.config = config
}

func (s *Snapshotter) TakeSnapshot(ctx context.Context) (time.Duration, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	snapshot, err := os.CreateTemp("", "snapshot")
	if err != nil {
		return s.config.Frequency, err
	}

	defer os.Remove(snapshot.Name())

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	err = s.client.TakeSnapshot(ctx, snapshot)
	if err != nil {
		return s.config.Frequency, err
	}

	_, err = snapshot.Seek(0, io.SeekStart)
	if err != nil {
		return s.config.Frequency, err
	}

	return s.config.Frequency, s.uploadSnapshot(ctx, snapshot, time.Now().Format(s.config.TimestampFormat))
}

func (s *Snapshotter) uploadSnapshot(ctx context.Context, snapshot io.Reader, timestamp string) error {
	var errs error
	for _, uploader := range s.uploaders {
		err := uploader.Upload(ctx, snapshot, s.config.NamePrefix, timestamp, s.config.NameSuffix, s.config.Retain)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("unable to upload snapshot: %s", err))
		} else {
			log.Printf("Successfully uploaded snapshot to %s\n", uploader.Destination())
		}
	}

	return errs
}
