package store

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

// TestChannelAllowedUsersMigrationAndRoundTrip plants a channels table
// from before allowed_users existed, migrates it, and checks that the
// old bot stays open while a new allowlist survives save → load.
func TestChannelAllowedUsersMigrationAndRoundTrip(t *testing.T) {
	st, err := NewDBStore("sqlite", "file:"+filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	ctx := context.Background()

	if _, err := st.DB().ExecContext(ctx, `CREATE TABLE channels (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL DEFAULT '',
		agent_id TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL,
		account_id TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		bot_token TEXT NOT NULL DEFAULT '',
		base_url TEXT NOT NULL DEFAULT '',
		platform_user_id TEXT NOT NULL DEFAULT '',
		shared_identity INTEGER NOT NULL DEFAULT 0,
		data TEXT NOT NULL DEFAULT '{}',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (type, account_id)
	)`); err != nil {
		t.Fatalf("plant old table: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO channels (id, user_id, agent_id, type, account_id) VALUES ('ch_old', 'u_1', 'agt_1', 'telegram', 'old_bot')`); err != nil {
		t.Fatalf("plant old row: %v", err)
	}

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	old, err := st.LookupChannel(ctx, "telegram", "old_bot")
	if err != nil {
		t.Fatalf("lookup old bot: %v", err)
	}
	if len(old.AllowedUsers) != 0 {
		t.Fatalf("pre-existing bot must stay open, got allowlist %v", old.AllowedUsers)
	}

	old.AllowedUsers = []string{"111", "222"}
	if err := st.SaveChannel(ctx, old); err != nil {
		t.Fatalf("save allowlist: %v", err)
	}
	listed, err := st.ListChannels(ctx, "u_1", "agt_1")
	if err != nil || len(listed) != 1 {
		t.Fatalf("list channels: %v, %d rows", err, len(listed))
	}
	if !slices.Equal(listed[0].AllowedUsers, []string{"111", "222"}) {
		t.Fatalf("allowlist round-trip = %v", listed[0].AllowedUsers)
	}

	old.AllowedUsers = nil
	if err := st.SaveChannel(ctx, old); err != nil {
		t.Fatalf("clear allowlist: %v", err)
	}
	cleared, err := st.LookupChannel(ctx, "telegram", "old_bot")
	if err != nil {
		t.Fatalf("lookup cleared: %v", err)
	}
	if len(cleared.AllowedUsers) != 0 {
		t.Fatalf("cleared allowlist = %v, want empty", cleared.AllowedUsers)
	}
}
