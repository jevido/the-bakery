package infra

import (
	"context"

	"github.com/jevido/the-bakery/services/api/contexts/agents/app"
)

type Shares struct{}

// Add is idempotent: sharing twice changes nothing.
func (Shares) Add(ctx context.Context, agentID, guildID uint64) error {
	_, err := query(ctx).Exec(`INSERT INTO agent_shares (agent_id, guild_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, agentID, guildID)
	return err
}

func (Shares) Remove(ctx context.Context, agentID, guildID uint64) error {
	_, err := query(ctx).Exec(`DELETE FROM agent_shares WHERE agent_id = ? AND guild_id = ?`, agentID, guildID)
	return err
}

func (Shares) GuildsOf(ctx context.Context, agentID uint64) ([]uint64, error) {
	var ids []uint64
	err := query(ctx).Table("agent_shares").Where("agent_id", agentID).OrderBy("guild_id").Pluck("guild_id", &ids)
	return ids, err
}

func (Shares) AgentsIn(ctx context.Context, guildID uint64) ([]uint64, error) {
	var ids []uint64
	err := query(ctx).Table("agent_shares").Where("guild_id", guildID).OrderBy("shared_at").Pluck("agent_id", &ids)
	return ids, err
}

var _ app.Shares = Shares{}
