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
	"mime"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/andreypremiere/jwtmanager"
	"github.com/google/uuid"
)

var (
	roomIDRegexLen1 = regexp.MustCompile(`^[a-zA-Z]$`)
	roomIDRegex     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*[a-zA-Z0-9]$`)
)

type AccountService struct {
	accountRepo   repositories.AccountRepository
	emailProvider *utils.MailService
	passHasher    *utils.PasswordHasher
	jwtManager    *jwtmanager.JWTManager
	s3Manager     *S3Manager
}

func (as *AccountService) UpdateRoomBio(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingTextFieldRequest) error {
	count := utf8.RuneCountInString(req.Value)

	if count > 1000 {
		return apperrors.ErrInvalidInput
	}

	err := as.accountRepo.UpdateRoomBio(ctx, roomId, req.Value)
	if err != nil {
		return err
	}

	return nil
}

func (as *AccountService) UpdateRoomName(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingTextFieldRequest) error {
	count := utf8.RuneCountInString(req.Value)

	if count == 0 {
		return apperrors.ErrInvalidInput
	}

	if count > 100 {
		return apperrors.ErrInvalidInput
	}

	return as.accountRepo.UpdateRoomName(ctx, roomId, req.Value)
}

func (as *AccountService) validateRoomUniqueId(input string) error {
	inputLength := utf8.RuneCountInString(input)

	if inputLength == 0 {
		return apperrors.ErrInvalidInput
	}

	if inputLength > 100 {
		return apperrors.ErrInvalidInput
	}

	if inputLength == 1 {
		if !roomIDRegexLen1.MatchString(input) {
			return apperrors.ErrInvalidInput
		}
		return nil
	}

	if !roomIDRegex.MatchString(input) {
		return apperrors.ErrInvalidInput
	}

	return nil
}

func (as *AccountService) UpdateRoomUniqueId(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingTextFieldRequest) error {
	
	err := as.validateRoomUniqueId(req.Value)
	if err != nil {
		return err
	}

	err = as.accountRepo.UpdateRoomUniqueId(ctx, roomId, req.Value)
	if err != nil {
		return err
	}

	return nil
}

func (as *AccountService) UpdateBackground(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingBackgroundRequest) (*responses.UpdatingBackgroundResponse, error) {
	// Ищем текущий путь фона в БД
	currentShortPath, err := as.accountRepo.GetBackgroundPath(ctx, roomId)
	if err != nil {
		return nil, err
	}

	var uploadURL string
	var targetShortPath string

	// Если фон уже есть — переиспользуем ключ для перезаписи
	if currentShortPath != "" {
		targetShortPath = currentShortPath
		uploadURL, err = as.s3Manager.GetPresignedUploadURLByPath(ctx, targetShortPath, 15*time.Minute)
		if err != nil {
			return nil, apperrors.ErrInternal
		}
	} else {
		// Если фона нет — генерируем новое имя файла по MIME-типу
		extensions, err := mime.ExtensionsByType(req.MimeType)
		var ext string

		if err == nil && len(extensions) > 0 {
			ext = extensions[0]
		} else {
			ext = ".jpg"
		}
		fileId := uuid.New().String()

		// Генерируем новую ссылку для загрузки
		uploadURL, targetShortPath, err = as.s3Manager.GetPresignedUploadURL(ctx, roomId.String(), fileId, ext, 15*time.Minute)
		if err != nil {
			return nil, apperrors.ErrInternal
		}

		// Сохраняем новый путь в базу данных
		err = as.accountRepo.UpdateBackgroundPath(ctx, roomId, targetShortPath)
		if err != nil {
			return nil, err
		}
	}

	//  Формируем публичную ссылку для отображения
	publicURL := as.s3Manager.FormatFullURL(targetShortPath)

	return &responses.UpdatingBackgroundResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
	}, nil
}

func (as *AccountService) UpdateAvatar(ctx context.Context, roomId uuid.UUID, req *requests.UpdatingAvatarRequest) (*responses.UpdatingAvatarResponse, error) {
	currentShortPath, err := as.accountRepo.GetAvatarPath(ctx, roomId)
	if err != nil {
		return nil, err
	}

	var uploadURL string
	var targetShortPath string

	// Проверяем, существует ли уже ключ в хранилище
	if currentShortPath != "" {
		// Ключ есть! Генерируем PUT-ссылку для перезаписи старого файла
		targetShortPath = currentShortPath
		uploadURL, err = as.s3Manager.GetPresignedUploadURLByPath(ctx, targetShortPath, 15*time.Minute)
		if err != nil {
			return nil, apperrors.ErrInternal
		}
	} else {
		extensions, err := mime.ExtensionsByType(req.MimeType)
		var ext string

		if err == nil && len(extensions) > 0 {
			ext = extensions[0] 
		} else {
			ext = ".jpg"
		}
		fileId := uuid.New().String()

		uploadURL, targetShortPath, err = as.s3Manager.GetPresignedUploadURL(ctx, roomId.String(), fileId, ext, 15*time.Minute)
		if err != nil {
			return nil, apperrors.ErrInternal
		}

		err = as.accountRepo.UpdateAvatarPath(ctx, roomId, targetShortPath)
		if err != nil {
			return nil, err
		}
	}

	publicURL := as.s3Manager.FormatFullURL(targetShortPath)

	return &responses.UpdatingAvatarResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
	}, nil
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

func (s *AccountService) SetConfigured(ctx context.Context, userID uuid.UUID) error {

	err := s.accountRepo.SetConfigured(ctx, userID)
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

func (as *AccountService) SearchRooms(ctx context.Context, page int, limit int, value string) (*responses.FoundRooms, error) {
	offset := limit * page

	rooms, err := as.accountRepo.SearchRooms(ctx, limit, offset, value)
	if err != nil {
		return nil, err
	}

	if len(rooms) == 0 {
		return &responses.FoundRooms{Rooms: make([]*responses.RoomInfoExpanded, 0)}, nil
	}

	roomsList := make([]*responses.RoomInfoExpanded, 0, len(rooms))

	for _, item := range rooms {
		roomsList = append(roomsList, &responses.RoomInfoExpanded{
			Id:           item.ID,
			RoomUniqueId: item.RoomUniqueID,
			Nickname:     item.RoomName,
			AvatarURL:    as.s3Manager.FormatFullURL(item.AvatarURL),
		})
	}

	return &responses.FoundRooms{Rooms: roomsList}, nil
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
