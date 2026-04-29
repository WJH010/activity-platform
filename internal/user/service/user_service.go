package service

import (
	"activity-platform/internal/config"
	msgsvc "activity-platform/internal/message/service"
	"activity-platform/internal/user/dto"
	"activity-platform/internal/user/model"
	"activity-platform/internal/user/repository"
	"activity-platform/internal/utils"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

// WxLoginResponse 微信登录请求参数
type WxLoginResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
	ErrCode    int    `json:"errcode,omitempty"`
	ErrMsg     string `json:"errmsg,omitempty"`
}

// UserService 用户服务接口
type UserService interface {
	Login(ctx context.Context, code string, encryptedData string, iv string) (string, string, string, error)
	UpdateUserInfo(ctx context.Context, userID int, req dto.UserUpdateRequest) error
	GetUserByID(ctx context.Context, userID int) (*dto.UserInfoResponse, error)
	ListAllUsers(ctx context.Context, page, pageSize int, req dto.ListUsersRequest) ([]*dto.ListUsersResponse, int64, error)
	// CreateAdminUser 新增管理员
	CreateAdminUser(ctx context.Context, req dto.CreateAdminRequest, operator int) error
	// BgLogin 后台登录
	BgLogin(ctx context.Context, req dto.BgLoginRequest) (string, string, error)
	// UpdateAdminUser 更新管理员
	UpdateAdminUser(ctx context.Context, userID int, req dto.UpdateAdminRequest, operator int) error
	// UpdateAdminStatus 更新管理员状态
	UpdateAdminStatus(ctx context.Context, userID int, Operation string, operator int) error
	// RefreshToken 刷新access token
	RefreshToken(ctx context.Context, refreshToken string, sessionSign string) (string, string, error)
	// BgRefreshToken 通过刷新令牌获取新的access token和refresh token
	BgRefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	// WxLogout 退出登录
	WxLogout(ctx context.Context, userID int) error
}

// UserServiceImpl 用户服务实现
type UserServiceImpl struct {
	userRepo repository.UserRepository
	msgSvc   msgsvc.MsgGroupService
	cfg      *config.Config
}

// Argon2参数配置
const (
	// 内存成本：哈希过程中使用的内存量（字节）
	argonMemory uint32 = 65536 // 64MB
	// 时间成本：计算迭代次数
	argonTime uint32 = 3
	// 并行度：使用的CPU核心数
	argonThreads uint8 = 4
	// 生成的哈希长度（字节）
	argonKeyLen uint32 = 32
	// 盐值长度（字节）
	argonSaltLen uint32 = 16
)

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, msgSvc msgsvc.MsgGroupService, cfg *config.Config) UserService {
	return &UserServiceImpl{userRepo: userRepo, msgSvc: msgSvc, cfg: cfg}
}

// Login 微信登录逻辑
func (svc *UserServiceImpl) Login(ctx context.Context, code string, encryptedData string, iv string) (string, string, string, error) {
	// 调用微信接口
	wxResp, err := svc.getFromWechat(code)
	if err != nil {
		return "", "", "", err
	}

	// 查找或创建用户
	userID, userRole, err := svc.findOrCreateUser(ctx, wxResp.OpenID, wxResp.SessionKey, wxResp.UnionID, encryptedData, iv)
	if err != nil {
		return "", "", "", err
	}

	// 生成access token
	accessToken, err := svc.generateAccessToken(wxResp.OpenID, userID, userRole)
	if err != nil {
		return "", "", "", err
	}
	// 生成refresh token
	refreshToken, err := svc.generateRefreshToken(userID)
	if err != nil {
		return "", "", "", err
	}
	// 存储refresh token到数据库
	err = svc.userRepo.UpdateRefreshToken(ctx, userID, refreshToken)
	if err != nil {
		return "", "", "", err
	}
	// 生成session sign会话凭证
	sessionSign := hmacSha256(fmt.Sprintf("%s%s", wxResp.SessionKey, wxResp.OpenID), svc.cfg.JWT.SessionKeySecret)

	return accessToken, refreshToken, sessionSign, nil
}

// getFromWechat 调用微信接口
func (svc *UserServiceImpl) getFromWechat(code string) (WxLoginResponse, error) {
	var wxResp WxLoginResponse

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", svc.cfg.Wechat.AppID, svc.cfg.Wechat.AppSecret, code)

	resp, err := http.Get(url)
	if err != nil {
		return wxResp, err
	}
	defer resp.Body.Close()

	// 读取微信响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return wxResp, utils.NewSystemError(fmt.Errorf("读取微信响应失败: %v", err))
	}

	// 解析微信响应
	err = json.Unmarshal(body, &wxResp)
	if err != nil {
		return wxResp, utils.NewSystemError(fmt.Errorf("解析微信响应失败: %v", err))
	}

	if wxResp.ErrCode != 0 {
		return wxResp, utils.NewSystemError(fmt.Errorf("微信登录错误: %d - %s", wxResp.ErrCode, wxResp.ErrMsg))
	}

	return wxResp, nil
}

// 查找或创建用户
func (svc *UserServiceImpl) findOrCreateUser(ctx context.Context, openID, sessionKey, unionID, encryptedData, iv string) (int, string, error) {
	// 查找用户
	user, err := svc.userRepo.GetUserByOpenID(ctx, openID)
	if err != nil {
		return 0, "", err
	}

	now := time.Now()

	// 解密手机号
	phoneNumberJSON, err := decryptPhoneNumber(sessionKey, svc.cfg.Wechat.AppID, encryptedData, iv)
	if err != nil {
		return 0, "", err
	}
	countryCode := phoneNumberJSON["countryCode"].(string)
	phoneNumber := phoneNumberJSON["phoneNumber"].(string)

	// 如果用户不存在，创建新用户
	if user == nil {
		// 生成默认昵称和头像
		defaultNickname := fmt.Sprintf("微信用户%s", openID[len(openID)-5:]) // 拼接OpenID的后5位作为默认昵称
		defaultAvatar := "http://47.113.194.28:9000/news-platform/images/202508/1754126743005963551.webp"
		user = &model.User{
			OpenID:        openID,
			SessionKey:    sessionKey,
			UnionID:       unionID,
			Nickname:      defaultNickname,
			AvatarURL:     defaultAvatar,
			CountryCode:   countryCode,
			PhoneNumber:   phoneNumber,
			LastLoginTime: now,
		}

		if err := svc.userRepo.Create(ctx, user); err != nil {
			return user.UserID, user.Role, err
		}
		// 新用户创建成功后加入全体成员的群组
		svc.msgSvc.AddUserToAllUserGroups(ctx, user.UserID)
	} else {
		// 如果用户存在，更新session_key、手机号和登录时间
		if err := svc.userRepo.UpdateSessionAndLoginTime(ctx, user.UserID, sessionKey, countryCode, phoneNumber); err != nil {
			return 0, "", err
		}
	}

	return user.UserID, user.Role, nil
}

// 生成JWT Token（目前只用于管理平台的登录）
func (svc *UserServiceImpl) generateToken(openID string, userID int, userRole string) (string, error) {
	// 创建令牌声明
	claims := jwt.MapClaims{
		"openid":    openID,
		"userid":    userID,
		"user_role": userRole,
		"exp":       time.Now().Add(time.Hour * 24).Unix(), // 令牌有效期1天
		"iat":       time.Now().Unix(),
		"type":      "access",
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenStr, err := token.SignedString([]byte(svc.cfg.JWT.JwtSecret))
	if err != nil {
		return "", utils.NewSystemError(fmt.Errorf("生成Token失败: %w", err))
	}
	return tokenStr, nil
}

// 微信小程序端登录，采用双token机制（access token和refresh token）
// 生成access token（短期有效）
func (svc *UserServiceImpl) generateAccessToken(openID string, userID int, userRole string) (string, error) {
	claims := jwt.MapClaims{
		"openid":    openID,
		"userid":    userID,
		"user_role": userRole,
		"exp":       time.Now().Add(time.Hour * 24).Unix(), // 1天
		"iat":       time.Now().Unix(),
		"type":      "access", // 标记令牌类型
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenstr, err := token.SignedString([]byte(svc.cfg.JWT.JwtSecret))
	if err != nil {
		return "", utils.NewSystemError(fmt.Errorf("生成access token失败: %w", err))
	}
	return tokenstr, nil
}

// 生成refresh token（长期有效）
func (svc *UserServiceImpl) generateRefreshToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"userid": userID,
		"exp":    time.Now().Add(time.Hour * 168).Unix(), // 2周
		"iat":    time.Now().Unix(),
		"type":   "refresh", // 标记令牌类型
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenstr, err := token.SignedString([]byte(svc.cfg.JWT.RefreshSecret))
	if err != nil {
		return "", utils.NewSystemError(fmt.Errorf("生成refresh token失败: %w", err))
	}
	return tokenstr, nil
}

// RefreshClaims 自定义Claims结构体，明确指定字段类型
type RefreshClaims struct {
	UserID             int    `json:"userid"`
	Type               string `json:"type"`
	jwt.StandardClaims        // 嵌入标准声明
}

// RefreshToken 通过刷新令牌获取新的access token和refresh token
func (svc *UserServiceImpl) RefreshToken(ctx context.Context, refreshToken string, sessionSign string) (string, string, error) {
	// 解析refresh token
	claims, err := svc.parseRefreshToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	// 验证用户存在性
	userID := claims.UserID
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	// 验证refresh token是否匹配
	if user.RefreshToken != refreshToken {
		return "", "", utils.NewBusinessError(utils.ErrCodeRefreshTokenExpired, "认证信息已失效，请重新登录")
	}

	// 验证会话凭证是否匹配
	expectedSessionSign := hmacSha256(fmt.Sprintf("%s%s", user.SessionKey, user.OpenID), svc.cfg.JWT.SessionKeySecret)
	if sessionSign != expectedSessionSign {
		return "", "", utils.NewBusinessError(utils.ErrCodeAuthFailed, "会话凭证不匹配")
	}

	// 生成新的access token
	accessToken, err := svc.generateAccessToken(user.OpenID, userID, user.Role)
	if err != nil {
		return "", "", err
	}
	// 同时生成新的refresh token，实现滚动更新
	newRefreshToken, err := svc.generateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}
	// 存储新的refresh token到数据库
	err = svc.userRepo.UpdateRefreshToken(ctx, userID, newRefreshToken)
	if err != nil {
		return "", "", err
	}
	return accessToken, newRefreshToken, nil
}

// hmacSha256 计算HMAC-SHA256哈希值生成会话凭证
func hmacSha256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// WxLogout 微信小程序登出（失效刷新令牌）
func (svc *UserServiceImpl) WxLogout(ctx context.Context, userID int) error {
	return svc.userRepo.UpdateRefreshToken(ctx, userID, "")
}

// parseRefreshToken 解析refresh token
func (svc *UserServiceImpl) parseRefreshToken(tokenString string) (*RefreshClaims, error) {
	secret := []byte(svc.cfg.JWT.RefreshSecret)

	// 解析令牌时指定自定义Claims
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, utils.NewSystemError(fmt.Errorf("无效的签名方法"))
		}
		return secret, nil
	})

	if err != nil {
		var validationErr *jwt.ValidationError
		if errors.As(err, &validationErr) {
			if validationErr.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, utils.NewBusinessError(utils.ErrCodeRefreshTokenExpired, "认证信息已失效，请重新登录")
			}
		}
		return nil, err
	}

	// 验证令牌有效性并转换为自定义Claims
	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		// 检查令牌类型
		if claims.Type != "refresh" {
			return nil, utils.NewBusinessError(utils.ErrCodeTokenTypeInvalid, "无效的token类型")
		}
		return claims, nil
	}

	return nil, utils.NewSystemError(fmt.Errorf("无效的token"))
}

// UpdateUserInfo 更新用户信息
func (svc *UserServiceImpl) UpdateUserInfo(ctx context.Context, userID int, req dto.UserUpdateRequest) error {
	// 查询用户是否存在
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在，请刷新后重试")
	}

	// 构建更新字段映射
	updateFields := make(map[string]interface{})
	if req.Nickname != nil {
		updateFields["nickname"] = *req.Nickname
	}
	if req.AvatarURL != nil {
		updateFields["avatar_url"] = *req.AvatarURL
	}
	if req.Name != nil {
		updateFields["name"] = *req.Name
	}
	if req.Gender != nil {
		updateFields["gender"] = *req.Gender
	}
	// if req.PhoneNumber != nil {
	// 	updateFields["phone_number"] = *req.PhoneNumber
	// }
	if req.Email != nil {
		updateFields["email"] = *req.Email
	}
	if req.Unit != nil {
		updateFields["unit"] = *req.Unit
	}
	if req.Department != nil {
		updateFields["department"] = *req.Department
	}
	if req.Position != nil {
		updateFields["position"] = *req.Position
	}
	if req.Industry != nil {
		updateFields["industry"] = *req.Industry
	}

	// 执行更新
	if len(updateFields) > 0 {
		if err := svc.userRepo.Update(ctx, userID, updateFields); err != nil {
			return err
		}
	}

	return nil
}

func (svc *UserServiceImpl) GetUserByID(ctx context.Context, userID int) (*dto.UserInfoResponse, error) {
	// 查询用户信息
	user, err := svc.userRepo.GetUserInfoByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在，请刷新后重试")
	}

	return user, nil
}

// ListAllUsers 分页查询用户列表
func (svc *UserServiceImpl) ListAllUsers(ctx context.Context, page, pageSize int, req dto.ListUsersRequest) ([]*dto.ListUsersResponse, int64, error) {
	return svc.userRepo.ListAllUsers(ctx, page, pageSize, req)
}

// CreateAdminUser 新增管理员
func (svc *UserServiceImpl) CreateAdminUser(ctx context.Context, req dto.CreateAdminRequest, operator int) error {
	var avatar string
	if req.AvatarURL == "" {
		avatar = "http://47.113.194.28:9000/news-platform/images/202508/1754126743005963551.webp"
	} else {
		avatar = req.AvatarURL
	}

	// 对密码进行哈希处理
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return err
	}

	// 创建数据
	user := &model.User{
		Nickname:      req.Nickname,
		Name:          req.Name,
		AvatarURL:     avatar,
		PhoneNumber:   req.PhoneNumber,
		Email:         req.Email,
		Role:          req.Role,
		Password:      hashedPassword,
		LastLoginTime: time.Now(),
		CreateUser:    operator,
		UpdateUser:    operator,
	}

	if err := svc.userRepo.Create(ctx, user); err != nil {
		return err
	}
	return nil
}

// UpdateAdminUser 更新管理员
func (svc *UserServiceImpl) UpdateAdminUser(ctx context.Context, userID int, req dto.UpdateAdminRequest, operator int) error {
	// 查询用户是否存在
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在，请刷新后重试")
	}

	// 构建更新字段映射
	updateFields := make(map[string]interface{})
	if req.Nickname != nil {
		updateFields["nickname"] = *req.Nickname
	}
	if req.AvatarURL != nil {
		updateFields["avatar_url"] = *req.AvatarURL
	}
	if req.Name != nil {
		updateFields["name"] = *req.Name
	}
	if req.Role != nil {
		updateFields["role"] = *req.Role
	}
	if req.Email != nil {
		updateFields["email"] = *req.Email
	}
	if req.Password != nil {
		// 密码需要哈希处理
		hashedPassword, err := hashPassword(*req.Password)
		if err != nil {
			return err
		}
		updateFields["password"] = hashedPassword
	}
	updateFields["update_user"] = operator

	// 执行更新
	if len(updateFields) > 0 {
		if err := svc.userRepo.Update(ctx, userID, updateFields); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAdminStatus 禁用/启用管理员账号
func (svc *UserServiceImpl) UpdateAdminStatus(ctx context.Context, userID int, Operation string, operator int) error {
	// 查询用户是否存在
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return utils.NewBusinessError(utils.ErrCodeResourceNotFound, "用户不存在，请刷新后重试")
	}
	if Operation == "DISABLE" {
		if user.Status == utils.UserStatusDisabled {
			return utils.NewBusinessError(utils.ErrCodeResourceConflict, "用户已被禁用，请勿重复操作")
		}
	} else if Operation == "ENABLE" {
		if user.Status == utils.UserStatusEnabled {
			return utils.NewBusinessError(utils.ErrCodeResourceConflict, "用户已被启用，请勿重复操作")
		}
	} else {
		return utils.NewBusinessError(utils.ErrCodeParamInvalid, "操作类型错误")
	}

	// 构建更新字段映射
	updateFields := make(map[string]any)
	if Operation == "DISABLE" {
		updateFields["status"] = utils.UserStatusDisabled
	} else if Operation == "ENABLE" {
		updateFields["status"] = utils.UserStatusEnabled
	}
	updateFields["update_user"] = operator

	// 执行更新
	if len(updateFields) > 0 {
		if err := svc.userRepo.Update(ctx, userID, updateFields); err != nil {
			return err
		}
	}
	return nil
}

// 生成密码哈希
func hashPassword(password string) (string, error) {
	// 生成随机盐值
	salt := make([]byte, argonSaltLen)
	_, err := rand.Read(salt)
	if err != nil {
		return "", utils.NewSystemError(fmt.Errorf("生成随机盐值失败: %w", err))
	}

	// 使用Argon2id变体进行哈希（推荐用于密码哈希）
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// 组合盐值和哈希值，并进行Base64编码以便存储
	// 格式: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 包含算法参数以便验证时使用
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash)

	return encodedHash, nil
}

// BgLogin 后台登录
func (svc *UserServiceImpl) BgLogin(ctx context.Context, req dto.BgLoginRequest) (string, string, error) {
	// 从数据库中根据手机号查询密码
	userInfo, err := svc.userRepo.GetPasswordByPhone(ctx, req.PhoneNumber)
	if err != nil {
		return "", "", err
	}
	// 微信用户密码为空，不允许登录后台系统
	if userInfo.Password == "" {
		return "", "", utils.NewBusinessError(utils.ErrCodeAuthFailed, "账号未设置密码，无法登录后台系统")
	}

	// 验证密码
	ok, err := verifyPassword(userInfo.Password, req.Password)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", utils.NewBusinessError(utils.ErrCodeAuthFailed, "密码错误")
	}

	// 检查用户状态
	if userInfo.Status != utils.UserStatusEnabled {
		return "", "", utils.NewBusinessError(utils.ErrCodeAuthFailed, "账号已被禁用，无法登录后台系统")
	}

	// 检查用户角色
	// if userInfo.Role != utils.RoleAdmin && userInfo.Role != utils.RoleSuperAdmin {
	// 	return "", "", utils.NewBusinessError(utils.ErrCodeAuthFailed, "非管理员角色，无法登录后台系统")
	// }

	// 更新最后登录时间
	updateFields := make(map[string]any)
	updateFields["last_login_time"] = time.Now()
	if len(updateFields) > 0 {
		if err := svc.userRepo.Update(ctx, userInfo.UserID, updateFields); err != nil {
			// return "", err
			// 只记录日志，不影响登录成功
			logrus.Errorf("更新用户[%d]最后登录时间失败: %v", userInfo.UserID, err)
		}
	}

	// 登录成功，生成JWT Token
	token, err := svc.generateToken(req.PhoneNumber, userInfo.UserID, userInfo.Role)
	if err != nil {
		return "", "", err
	}

	// 生成refresh token
	refreshToken, err := svc.generateRefreshToken(userInfo.UserID)
	if err != nil {
		return "", "", err
	}

	// 存储refresh token到数据库
	err = svc.userRepo.UpdateRefreshToken(ctx, userInfo.UserID, refreshToken)
	if err != nil {
		return "", "", err
	}

	return token, refreshToken, nil
}

// BgRefreshToken 通过刷新令牌获取新的access token和refresh token
func (svc *UserServiceImpl) BgRefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// 解析refresh token
	claims, err := svc.parseRefreshToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	// 验证用户存在性
	userID := claims.UserID
	user, err := svc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", err
	}

	// 验证refresh token是否匹配
	if user.RefreshToken != refreshToken {
		return "", "", utils.NewBusinessError(utils.ErrCodeRefreshTokenExpired, "认证信息已失效，请重新登录")
	}

	// 生成新的access token
	accessToken, err := svc.generateAccessToken(user.OpenID, userID, user.Role)
	if err != nil {
		return "", "", err
	}
	// 同时生成新的refresh token，实现滚动更新
	newRefreshToken, err := svc.generateRefreshToken(userID)
	if err != nil {
		return "", "", err
	}
	// 存储新的refresh token到数据库
	err = svc.userRepo.UpdateRefreshToken(ctx, userID, newRefreshToken)
	if err != nil {
		return "", "", err
	}
	return accessToken, newRefreshToken, nil
}

// 验证密码
func verifyPassword(encodedHash, password string) (bool, error) {
	// 解析格式: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	// 按 $ 分割字符串，得到各部分
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, utils.NewSystemError(fmt.Errorf("哈希格式错误"))
	}
	// 验证算法是否为 argon2id
	if parts[1] != "argon2id" {
		return false, utils.NewSystemError(fmt.Errorf("不支持的算法: %s", parts[1]))
	}

	// 解析版本号（如 v=19）
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, utils.NewSystemError(fmt.Errorf("解析版本失败: %v", err))
	}

	// 解析参数（m=内存, t=时间, p=并行度）
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, utils.NewSystemError(fmt.Errorf("解析参数失败: %v", err))
	}

	// 提取盐值和哈希值（直接从分割结果中获取，避免解析错误）
	saltStr := parts[4]
	hashStr := parts[5]

	// 解码盐值和哈希值
	saltBytes, err := base64.RawStdEncoding.DecodeString(saltStr)
	if err != nil {
		return false, utils.NewSystemError(fmt.Errorf("解码盐值失败: %w", err))
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hashStr)
	if err != nil {
		return false, utils.NewSystemError(fmt.Errorf("解码哈希值失败: %w", err))
	}

	// 使用相同的参数计算输入密码的哈希
	inputHash := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, uint32(len(hashBytes)))

	// 比较计算出的哈希和存储的哈希
	return constantTimeCompare(inputHash, hashBytes), nil
}

// 常量时间比较函数，防止时序攻击
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := range a {
		result |= int(a[i] ^ b[i])
	}
	return result == 0
}

// decryptPhoneNumber 解密微信加密数据获取手机号
func decryptPhoneNumber(sKey, appID, encryptedData, iv string) (map[string]interface{}, error) {
	// Base64解码
	sessionKey, err := base64.StdEncoding.DecodeString(sKey)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("sessionKey base64解码失败: %w", err))
	}

	encryptedDataBytes, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("encryptedData base64解码失败: %w", err))
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("iv base64解码失败: %w", err))
	}

	// 创建AES解密器
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("创建AES解密器失败: %w", err))
	}

	// CBC模式解密
	mode := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(encryptedDataBytes))
	mode.CryptBlocks(decrypted, encryptedDataBytes)

	// 去除填充
	decrypted, err = unpad(decrypted)
	if err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("去除填充失败: %w，encryptedData：%s, iv：%s, sessionKey：%s", err, encryptedData, iv, sKey))
	}

	// 解析JSON
	var result map[string]interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		return nil, utils.NewSystemError(fmt.Errorf("JSON解析失败: %w", err))
	}

	// 验证水印中的appid
	watermark, ok := result["watermark"].(map[string]interface{})
	if !ok {
		return nil, utils.NewBusinessError(utils.ErrCodeAuthFailed, "数据中不包含水印信息")
	}

	appid, ok := watermark["appid"].(string)
	if !ok || appid != appID {
		return nil, utils.NewBusinessError(utils.ErrCodeAuthFailed, "appid校验失败")
	}

	return result, nil
}

// unpad 去除PKCS#7填充
func unpad(s []byte) ([]byte, error) {
	if len(s) == 0 {
		return nil, utils.NewSystemError(fmt.Errorf("数据长度为空"))
	}

	// 最后一个字节表示填充长度
	padLength := int(s[len(s)-1])
	if padLength > len(s) {
		return nil, utils.NewSystemError(fmt.Errorf("填充长度无效: %d, 数据长度: %d, 数据原文：%s", padLength, len(s), string(s)))
	}

	return s[:len(s)-padLength], nil
}
