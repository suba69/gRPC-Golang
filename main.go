package main

import (
	"fmt"
	pb "grpc-microservices/.proto"
	"grpc-microservices/service_1/db_connect"
	"log"
	"net"

	"github.com/jackc/pgx/v4/pgxpool"

	createuser "grpc-microservices/service_1/user_service"

	"google.golang.org/grpc"
)

var DbPool *pgxpool.Pool

func main() {
	var err error
	db_connect.DbPool, err = db_connect.ConnectToDatabase()
	if err != nil {
		fmt.Printf("Failed to initialize the database connection: %v\n", err)
		return
	}

	listen, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	err = db_connect.InitializeMongoCollection()
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB collection: %v", err)
		return
	}

	db_connect.ConnectToRedis()

	server := grpc.NewServer()
	authService := &createuser.AuthService{}
	pb.RegisterAuthServiceServer(server, authService)
	fmt.Println("gRPC server is running on :50051")
	if err := server.Serve(listen); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}
