package services

import (
	apperrors "account-microservice/app-errors"
	"account-microservice/contracts/account/requests"
	"account-microservice/contracts/account/responses"
	"account-microservice/models"
	"account-microservice/repositories"
	"account-microservice/utils"
	"context"
	"fmt"
	"time"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
)

type AccountService struct {
	accountRepo   repositories.AccountRepository
	emailProvider *utils.MailService
	passHasher    *utils.PasswordHasher
	jwtManager    *jwtmanager.JWTManager
	s3Manager     *S3Manager
}

func (as *AccountService) GetRoomFollowers(ctx context.Context, roomId uuid.UUID, page int, limit int) ([]responses.RoomInfo, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	authors, err := as.accountRepo.GetFollowers(ctx, roomId, limit, offset)
	if err != nil {
		return nil, err
	}

	if authors == nil {
		return []responses.RoomInfo{}, nil
	}

	for id, room := range authors {
		authors[id].AvatarUrl = as.s3Manager.FormatFullURL(room.AvatarUrl)
	}

	return authors, nil
}

func (as *AccountService) GetRoomFollowing(ctx context.Context, roomId uuid.UUID, page int, limit int) ([]responses.RoomInfo, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	authors, err := as.accountRepo.GetFollowing(ctx, roomId, limit, offset)
	if err != nil {
		return nil, err
	}

	if authors == nil {
		return []responses.RoomInfo{}, nil
	}

	for id, room := range authors {
		authors[id].AvatarUrl = as.s3Manager.FormatFullURL(room.AvatarUrl)
	}

	return authors, nil
}

func (as *AccountService) GetRoomInfo(context context.Context, id uuid.UUID) (*responses.RoomInfo, error) {
	room, err := as.accountRepo.GetRoomInfo(context, id)
	if err != nil {
		return nil, err
	}

	room.AvatarUrl = as.s3Manager.FormatFullURL(room.AvatarUrl)

	return room, nil
}

func (s *AccountService) CheckSubscription(ctx context.Context, followerId, followingId uuid.UUID) (bool, error) {
    isFollowed, err := s.accountRepo.CheckSubscription(ctx, followerId, followingId)
    if err != nil {
        return false, err
    }
    return isFollowed, nil
}

func (s *AccountService) Follow(ctx context.Context, followerId, followingId uuid.UUID) error {

    err := s.accountRepo.AddSubscription(ctx, followerId, followingId)
    if err != nil {
        return err
    }
    return nil
}

func (s *AccountService) Unfollow(ctx context.Context, followerId, followingId uuid.UUID) error {
    err := s.accountRepo.RemoveSubscription(ctx, followerId, followingId)
    if err != nil {
        return err
    }
    return nil
}

func (as *AccountService) UpdateRoom(context context.Context, roomId uuid.UUID, request *requests.UpdateRoomRequest) (*responses.UpdateRoomResponse, error) {
	var response responses.UpdateRoomResponse

	if request.AvatarFileName != "" {
		presignedUrlAvatar, staticUrlAvatar, err := as.s3Manager.GenerateUploadContext(context, roomId.String(), request.AvatarFileName)
		if err != nil {
			return nil, apperrors.ErrGeneratingLinksForMedia
		}

		request.AvatarFileName = staticUrlAvatar
		response.PresignedUrlAvatar = presignedUrlAvatar
	}
	if request.BackgroundFileName != "" {
		presignedUrlBack, staticUrlBack, err := as.s3Manager.GenerateUploadContext(context, roomId.String(), request.BackgroundFileName)
		if err != nil {
			return nil, apperrors.ErrGeneratingLinksForMedia
		}

		request.BackgroundFileName = staticUrlBack
		response.PresignedUrlBackground = presignedUrlBack
	}

	err := as.accountRepo.UpdateRoom(context, roomId, request)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *AccountService) GetRoomsInfoBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]responses.RoomInfo, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]responses.RoomInfo), nil
	}

	result, err := s.accountRepo.GetRoomsInfoByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	for id, room := range result {
		room.AvatarUrl = s.s3Manager.FormatFullURL(room.AvatarUrl)
		result[id] = room
	}

	// Здесь можно добавить логику кеширования в Redis, если данные часто запрашиваются
	return result, nil
}

func (as *AccountService) GetRoom(context context.Context, roomId uuid.UUID) (*responses.RoomResponse, error) {
	room, err := as.accountRepo.GetRoom(context, roomId)
	if err != nil {
		return nil, err
	}

	if room.AvatarPath != "" {
		room.AvatarPath = as.s3Manager.FormatFullURL(room.AvatarPath)
	}

	if room.BackgroundPath != "" {
		room.BackgroundPath = as.s3Manager.FormatFullURL(room.BackgroundPath)
	}

	return room, nil
}

func (as *AccountService) VerifyCode(context context.Context, userVerify *requests.VerifyUser) (*responses.AuthResponse, error) {
	if userVerify.Code == "" || len(userVerify.Code) != 6 {
		return nil, apperrors.ErrInvalidInput
	}

	isConfigured, err := as.accountRepo.GetStatusConfigured(context, userVerify.UserId)
	if err != nil {
		return nil, err
	}

	roomId, err := as.accountRepo.GetRoomIdByUserId(context, userVerify.UserId)
	if err != nil {
		return nil, err
	}

	gotCode, err := as.accountRepo.GetOTPCode(context, userVerify.UserId)
	if err != nil {
		return nil, err
	}

	if gotCode != userVerify.Code {
		return nil, apperrors.ErrInvalidCode
	}

	accessToken, _ := as.jwtManager.Generate(userVerify.UserId.String(), roomId.String())
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	err = as.accountRepo.VerifyAndCreateSession(context, userVerify.UserId, refreshToken, userVerify.DeviceInfo, expiresAt)
	if err != nil {
		return nil, err
	}

	response := &responses.AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken, IsConfigured: isConfigured}

	return response, nil
}

func (as *AccountService) NewAccount(ctx context.Context, newUser *requests.CreatingAccount) (*uuid.UUID, error) {
	newUserId := uuid.New()
	newRoomId := uuid.New()

	if newUser.Email == "" {
		return nil, apperrors.ErrInvalidInput
	}
	if newUser.Password == "" {
		return nil, apperrors.ErrInvalidInput
	}

	// Добавить проверку требования пароля

	hashPassword, err := as.passHasher.HashPassword(newUser.Password)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	roomUniqueId := fmt.Sprintf("USER-%s", newRoomId)
	roomName := roomUniqueId

	err = as.accountRepo.NewAccount(ctx, newUser.Email, newUserId, newRoomId, roomUniqueId, roomName, hashPassword)
	if err != nil {
		return nil, err
	}

	as.GenerateAndSendCode(newUserId, newUser.Email)

	return &newUserId, nil
}

func (as *AccountService) GenerateAndSendCode(userId uuid.UUID, email string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := as.doGenerateAndSend(ctx, userId, email)
		if err != nil {
			fmt.Printf("[ASYNC ERROR] Ошибка для юзера %s: %v\n", userId, err)
		}
	}()
}

func (as *AccountService) LoginUser(ctx context.Context, loginReq *requests.LoginUser) (*models.BaseUser, error) {
	user, err := as.accountRepo.GetUserByEmail(ctx, loginReq.Email)
	if err != nil {
		return nil, err
	}

	isEqual := as.passHasher.ComparePassword(loginReq.Password, user.PasswordHash)
	if !isEqual {
		return nil, apperrors.ErrInvalidPassword
	}

	as.GenerateAndSendCode(user.ID, user.Email)

	return user, nil
}

func (as *AccountService) RefreshSession(ctx context.Context, oldRefreshToken string) (*responses.RefreshTokens, error) {
	session, err := as.accountRepo.GetSessionByToken(ctx, oldRefreshToken)
	if err != nil {
		return nil, err
	}

	if session.ExpiresAt.Before(time.Now()) {
		_ = as.accountRepo.DeleteRefreshToken(ctx, oldRefreshToken)
		return nil, apperrors.ErrSessionExpired
	}

	newAccessToken, _ := as.jwtManager.Generate(session.UserId.String(), session.RoomId.String())
	newRefreshToken := uuid.New().String()
	newExpiresAt := time.Now().Add(30 * 24 * time.Hour)

	err = as.accountRepo.UpdateRefreshToken(ctx, oldRefreshToken, newRefreshToken, newExpiresAt)
	if err != nil {
		return nil, err
	}

	return &responses.RefreshTokens{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (as *AccountService) doGenerateAndSend(ctx context.Context, userId uuid.UUID, email string) error {
	code, err := utils.GenerateCode()
	if err != nil {
		return apperrors.ErrInternal
	}

	if err := as.accountRepo.AddCodeWithTimeout(ctx, userId, code); err != nil {
		return err
	}

	if err := as.emailProvider.SendVerificationCode(email, code); err != nil {
		return apperrors.ErrEmailProvider
	}

	return nil
}

func (as *AccountService) Logout(ctx context.Context, refreshToken string) error {
	return as.accountRepo.DeleteRefreshToken(ctx, refreshToken)
}

func (as *AccountService) RepeatSendingCode(ctx context.Context, userID uuid.UUID) error {
	user, err := as.accountRepo.GetUserEmailByID(ctx, userID)
	if err != nil {
		return err
	}

	as.GenerateAndSendCode(userID, user.Email)
	return nil
}

func NewAccountService(
	accountRepo *repositories.AccountRepository,
	emailProvider *utils.MailService,
	passwordHasher *utils.PasswordHasher,
	jwtManager *jwtmanager.JWTManager,
	s3Manager *S3Manager,
) *AccountService {
	return &AccountService{
		accountRepo:   *accountRepo,
		emailProvider: emailProvider,
		passHasher:    passwordHasher,
		jwtManager:    jwtManager,
		s3Manager:     s3Manager,
	}
}
