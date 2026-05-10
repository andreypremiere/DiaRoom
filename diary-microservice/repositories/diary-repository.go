package repositories

import "github.com/jackc/pgx/v5/pgxpool"

type DiaryRepository struct {
	db *pgxpool.Pool
}

func NewDiaryRepository(db *pgxpool.Pool) *DiaryRepository {
	return &DiaryRepository{
		db: db,
	}
}