package db_connect

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ctx = context.Background()

func ConnectToRedis() (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return redisClient, nil
}

func UpdateDataInRedis(redisClient *redis.Client, key string) error {
	data, err := getAllDataFromPostgreSQL(DbPool)
	if err != nil {
		return err
	}

	dataToCache, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = redisClient.Set(ctx, key, dataToCache, 1*time.Minute).Err()
	return err
}

func getAllDataFromPostgreSQL(DbPool *pgxpool.Pool) ([]string, error) {
	query := "SELECT balance FROM users"

	rows, err := DbPool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []string

	for rows.Next() {
		var balance string

		if err := rows.Scan(&balance); err != nil {
			return nil, err
		}
		data = append(data, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}
