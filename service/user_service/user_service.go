package createuser

import (
	"context"
	"database/sql"
	"fmt"
	pb "grpc-microservices/.proto"
	"grpc-microservices/service/db_connect"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"golang.org/x/crypto/bcrypt"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc/metadata"
)

type AuthService struct {
	pb.UnimplementedAuthServiceServer
	RefreshToken string
}

func CreateToken(username string, role string) (string, string, error) {
	var secretKey = []byte("fdgdthr64y456y46u4thbrt67y4ukmstyjaeyr57i69dytkumjg4")

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["role"] = role
	claims["exp"] = time.Now().Add(10 * time.Minute).Unix()
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshClaims["username"] = username
	refreshClaims["role"] = role
	refreshClaims["exp"] = time.Now().Add(30 * 24 * time.Hour).Unix()
	refreshTokenString, err := refreshToken.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func ParseToken(refreshToken string) (string, string, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неподдерживаемый метод подписи")
		}
		return []byte("fdgdthr64y456y46u4thbrt67y4ukmstyjaeyr57i69dytkumjg4"), nil
	})
	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, usernameFound := claims["username"].(string)
		role, roleFound := claims["role"].(string)
		if !usernameFound || !roleFound {
			return "", "", fmt.Errorf("имя пользователя или роль не найдены в Refresh-токене")
		}

		return username, role, nil
	}

	return "", "", fmt.Errorf("refresh-токен не действителен")
}

func (s *AuthService) RegisterUser(ctx context.Context, req *pb.RegisterUserRequest) (*pb.RegisterUserResponse, error) {
	username := req.Username
	password := req.Password
	if username == "" || password == "" {
		return nil, fmt.Errorf("имя пользователя или пароль не может быть пустым")
	}
	username = strings.ToLower(username)

	var role string
	if password == "kgpojl;g3549ujrgnkneri3i4t34039" {
		role = "admin"
	} else {
		role = "user"
	}

	exists, err := db_connect.UserExists(username, db_connect.DbPool)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("пользователь с таким username уже существует")
	}
	if err := db_connect.CreateUserInDatabase(username, password, role, db_connect.DbPool); err != nil {
		return nil, err
	}
	accessToken, refreshToken, err := CreateToken(username, role)
	if err != nil {
		return nil, err
	}

	s.RefreshToken = refreshToken

	response := &pb.RegisterUserResponse{
		Message:      "Регистрация успешна",
		Token:        accessToken,
		RefreshToken: refreshToken,
	}
	return response, nil
}

func (s *AuthService) GetAdminUsers(ctx context.Context, req *pb.GetAdminUsersRequest) (*pb.GetAdminUsersResponse, error) {
	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	_, role, err := ParseToken(s.RefreshToken)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, fmt.Errorf("недостаточно прав")
	}

	users, err := db_connect.GetAdminUsers()
	if err != nil {
		return nil, err
	}
	response := &pb.GetAdminUsersResponse{
		Users: users,
	}
	return response, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	_, role, err := ParseToken(s.RefreshToken)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		return nil, fmt.Errorf("недостаточно прав")
	}

	err = db_connect.DeleteUser(req.Username, db_connect.DbPool)
	if err != nil {
		return nil, err
	}

	response := &pb.DeleteUserResponse{
		Message: "Пользователь успешно удален",
	}
	return response, nil
}

func isAuthenticated(username, password, role string) bool {
	var passwordHash string

	exists, err := db_connect.UserExists(username, db_connect.DbPool)
	if err != nil {
		log.Printf("Ошибка проверки существования пользователя: %v", err)
		return false
	}

	if !exists {
		log.Printf("Пользователь не найден: %s", username)
		return false
	}

	err = db_connect.DbPool.QueryRow(context.Background(), "SELECT password FROM users WHERE username = $1", username).Scan(&passwordHash)
	if err != nil {
		log.Printf("Ошибка получения хеша пароля пользователя.: %v", err)
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		log.Printf("Пароли не совпадают: %s", username)
		return false
	}

	return true
}

func (s *AuthService) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.LoginUserResponse, error) {
	username := req.Username
	password := req.Password

	var role string
	if password == "kgpojl;g3549ujrgnkneri3i4t34039" {
		role = "admin"
	} else {
		role = "user"
	}

	if isAuthenticated(username, password, role) {
		accessToken, refreshToken, err := CreateToken(username, role)
		if err != nil {
			return nil, err
		}

		newRefreshToken := refreshToken
		s.RefreshToken = newRefreshToken

		response := &pb.LoginUserResponse{
			Message:      "Вход успешен",
			Token:        accessToken,
			RefreshToken: newRefreshToken,
		}
		return response, nil
	}
	return nil, fmt.Errorf("ошибка аутентификации")
}

func (s *AuthService) Logout(ctx context.Context, req *pb.LogoutUserRequest) (*pb.LogoutUserResponse, error) {
	s.RefreshToken = ""

	response := &pb.LogoutUserResponse{
		Message: "Выход из аккаунта успешен",
	}
	return response, nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, req *pb.UserProfileRequest) (*pb.UserProfileResponse, error) {
	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	md := metadata.New(map[string]string{"refreshToken": s.RefreshToken})
	_ = metadata.NewOutgoingContext(ctx, md)

	username, _, err := ParseToken(s.RefreshToken)
	if err != nil {
		return nil, err
	}

	var userProfile pb.UserProfileResponse
	var createdAt sql.NullTime
	var deletedAt sql.NullTime

	err = db_connect.DbPool.QueryRow(context.Background(), "SELECT username, role, balance, created_at, deleted_at FROM users WHERE username = $1", username).
		Scan(&userProfile.Username, &userProfile.Role, &userProfile.Balance, &createdAt, &deletedAt)

	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		createdTimestamp, _ := ptypes.TimestampProto(createdAt.Time)
		userProfile.CreatedAt = createdTimestamp
	}

	if deletedAt.Valid {
		deletedTimestamp, _ := ptypes.TimestampProto(deletedAt.Time)
		userProfile.DeletedAt = deletedTimestamp
	}

	return &userProfile, nil
}

func (s *AuthService) GetBanks(ctx context.Context, req *pb.GetBanksRequest) (*pb.GetBanksResponse, error) {
	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	collection, err := db_connect.GetMongoCollection()
	if err != nil {
		return nil, err
	}

	bankName := req.BankName
	filter := bson.M{"name": bankName}

	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())

	var bankList []*pb.Bank
	for cur.Next(context.TODO()) {
		var bank pb.Bank
		if err := cur.Decode(&bank); err != nil {
			return nil, err
		}
		bankList = append(bankList, &pb.Bank{
			Name:    bank.Name,
			Link:    bank.Link,
			Address: bank.Address,
		})
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	response := &pb.GetBanksResponse{
		Banks: bankList,
	}
	return response, nil
}

func (s *AuthService) AddBalance(ctx context.Context, req *pb.AddBalanceRequest) (*pb.AddBalanceResponse, error) {
	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	if req.Amount <= 0 {
		return nil, fmt.Errorf("некорректная сумма для пополнения баланса")
	}

	if req.Amount > 100000000 {
		return nil, fmt.Errorf("сумма пополнения превышает максимально допустимое значение")
	}

	md := metadata.New(map[string]string{"refreshToken": s.RefreshToken})
	_ = metadata.NewOutgoingContext(ctx, md)

	username, _, err := ParseToken(s.RefreshToken)
	if err != nil {
		return nil, err
	}
	exists, err := db_connect.UserExists(username, db_connect.DbPool)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("пользователь не найден")
	}

	_, err = db_connect.DbPool.Exec(context.Background(), "UPDATE users SET balance = balance + $1 WHERE username = $2", req.Amount, username)
	if err != nil {
		return nil, err
	}
	response := &pb.AddBalanceResponse{
		Message: "Баланс успешно пополнен",
	}
	return response, nil
}

func (s *AuthService) CheckBalance(ctx context.Context, req *pb.CheckBalanceRequest) (*pb.CheckBalanceResponse, error) {

	if s.RefreshToken == "" {
		return nil, fmt.Errorf("пожалуйста зарегестируйтесь или войдите в аккаунт")
	}

	md := metadata.New(map[string]string{"refreshToken": s.RefreshToken})
	_ = metadata.NewOutgoingContext(ctx, md)

	username, _, err := ParseToken(s.RefreshToken)
	if err != nil {
		return nil, err
	}

	exists, err := db_connect.UserExists(username, db_connect.DbPool)

	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("пользователь не найден")
	}

	var balance float64
	err = db_connect.DbPool.QueryRow(context.Background(), "SELECT balance FROM users WHERE username = $1", username).Scan(&balance)
	if err != nil {
		return nil, err
	}
	response := &pb.CheckBalanceResponse{Balance: balance}
	return response, nil
}
