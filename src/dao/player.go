package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:generate mockery --name PlayerDao
type PlayerDao interface {
	Upsert(ctx context.Context, player *Player) (*Player, error)
	GetTopN(ctx context.Context, limit int) ([]*Player, error)
	Truncate(ctx context.Context) error
}

type playerDao struct {
	db *gorm.DB
}

type Player struct {
	ID        int     `gorm:"primaryKey"`
	ClientID  string  `gorm:"uniqueIndex:udx_cleint_id"`
	Score     float64 `gorm:"index"`
	CreatedAt time.Time
}

func NewPlayerDao(db *gorm.DB) (PlayerDao, error) {
	dao := &playerDao{
		db: db,
	}

	if err := dao.db.AutoMigrate(&Player{}); err != nil {
		return nil, err
	}

	return dao, nil
}

func (d *playerDao) Upsert(ctx context.Context, player *Player) (*Player, error) {
	// INSERT INTO "users" ("client_id","score","created_at")
	// VALUES ('player-truncate',1.000000,'2023-06-14 20:15:50.535')
	// ON CONFLICT ("client_id") DO UPDATE
	// SET "score"="excluded"."score" RETURNING "id"

	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"score"}),
	}).Create(&player).Error; err != nil {
		return nil, err
	}

	return player, nil
}

func (d *playerDao) GetTopN(ctx context.Context, limit int) ([]*Player, error) {
	users := []*Player{}
	if err := d.db.WithContext(ctx).Order("score desc").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (d *playerDao) Truncate(ctx context.Context) error {
	return d.db.WithContext(ctx).Exec("TRUNCATE TABLE players").Error
}
