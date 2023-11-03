package db_connect

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var DbPool *pgxpool.Pool

func ConnectToDatabase() (*pgxpool.Pool, error) {
	dbURL := "postgres://postgres:olsap.6699@localhost:5432/grpc-golang"
	DbPool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	return DbPool, nil
}

func CreateUserInDatabase(username, password, role string, DbPool *pgxpool.Pool) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = DbPool.Exec(context.Background(), "INSERT INTO users (username, password, role) VALUES ($1, $2, $3)", username, hashedPassword, role)
	return err
}

func UserExists(username string, DbPool *pgxpool.Pool) (bool, error) {
	var exists bool
	err := DbPool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func GetAdminUsers() ([]string, error) {
	rows, err := DbPool.Query(context.Background(), "SELECT username, role, password FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var username, password, role string
		if err := rows.Scan(&username, &role, &password); err != nil {
			return nil, err
		}
		users = append(users, username, role, password)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func DeleteUser(username string, DbPool *pgxpool.Pool) error {
	query := "DELETE FROM users WHERE username = $1"
	_, err := DbPool.Exec(context.Background(), query, username)
	if err != nil {
		return fmt.Errorf("ошибка при удалении пользователя: %v", err)
	}
	return nil
}
