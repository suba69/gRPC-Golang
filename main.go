package main

import (
	"fmt"
	pb "grpc-microservices/.proto"
	"grpc-microservices/service_1/db_connect"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	createuser "grpc-microservices/service_1/user_service"

	"google.golang.org/grpc"
)

var DbPool *pgxpool.Pool

func main() {
	var err error
	db_connect.DbPool, err = db_connect.ConnectToDatabase()
	if err != nil {
		fmt.Printf("Не удалось инициализировать соединение с базой данных: %v\n", err)
		return
	}

	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Ошибка запуска: %v\n", err)
		return
	}

	err = db_connect.InitializeMongoCollection()
	if err != nil {
		log.Fatalf("Не удалось инициализировать коллекцию MongoDB: %v", err)
		return
	}

	redisClient, err := db_connect.ConnectToRedis()
	if err != nil {
		fmt.Printf("Не удалось инициализировать Redis: %v\n", err)
		return
	}

	err = db_connect.UpdateDataInRedis(redisClient, "cached_data")
	if err != nil {
		fmt.Printf("Не удалось обновить данные в Redis: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			time.Sleep(1 * time.Second)
			err := db_connect.UpdateDataInRedis(redisClient, "cached_data")
			if err != nil {
				fmt.Printf("Ошибка обновлений в Redis: %v\n", err)
			}
		}
	}()

	wg.Wait()

	server := grpc.NewServer()
	authService := &createuser.AuthService{}
	pb.RegisterAuthServiceServer(server, authService)
	fmt.Println("gRPC сервер запущен на:50051")
	if err := server.Serve(listen); err != nil {
		fmt.Printf("Не удалось обслужить: %v\n", err)
	}
}
