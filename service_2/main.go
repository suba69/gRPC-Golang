package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ctx = context.Background()

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	dbURL := "postgres://postgres:olsap.6699@localhost:5432/grpc-golang"

	DbPool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Println("Ошибка при подключении к базе данных:", err)
		return
	}

	key := "cached_data"

	cachedData, err := redisClient.Get(ctx, key).Result()
	if err == nil {
		var dataFromSource []string
		if err := json.Unmarshal([]byte(cachedData), &dataFromSource); err != nil {
			fmt.Println("Ошибка при декодировании данных из Redis:", err)
			return
		}
		fmt.Println("Данные из Redis-кеша:", dataFromSource)
	} else if err == redis.Nil {
		fmt.Println("Данные отсутствуют в Redis-кеше")

		dataFromSource, err := getAllDataFromPostgreSQL(DbPool)
		if err != nil {
			fmt.Println("Ошибка при получении данных из PostgreSQL:", err)
			return
		}

		dataToCache, err := json.Marshal(dataFromSource)
		if err != nil {
			fmt.Println("Ошибка при сериализации данных для кеширования:", err)
			return
		}

		err = redisClient.Set(ctx, key, dataToCache, 1*time.Minute).Err()

		if err != nil {
			fmt.Println("Ошибка при сохранении данных в Redis-кеш:", err)
		} else {
			fmt.Println("Данные сохранены в Redis-кеше:", dataFromSource)
		}
	} else {
		fmt.Println("Ошибка при получении данных из Redis:", err)
	}
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
