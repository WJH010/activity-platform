package middleware

import (
	"activity-platform/internal/config"
	rd "activity-platform/internal/redis"
	"activity-platform/internal/utils"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// CustomClaims 自定义Claims结构体，明确指定字段类型
type CustomClaims struct {
	OpenID             string `json:"openid"`
	UserID             int    `json:"userid"`
	UserRole           string `json:"user_role"`
	Type               string `json:"type"`
	jwt.StandardClaims        // 嵌入标准声明
}

// RoleMiddleware 角色权限认证中间件，用于校验角色权限
func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从上下文获取用户角色
		role, exists := ctx.Get("user_role")
		if !exists {
			utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeResourceNotFound, "未获取到用户角色信息"))
			ctx.Abort()
			return
		}

		// 类型断言，确保角色是字符串类型
		userRoleStr, ok := role.(string)
		if !ok {
			utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodeInvalidRole, "用户角色格式无效"))
			ctx.Abort()
			return
		}

		// 检查权限: 用户角色是否有权访问所需角色的接口
		if !utils.HasAccess(userRoleStr, requiredRole) {
			utils.HandlerFunc(ctx, utils.NewBusinessError(utils.ErrCodePermissionDenied, "没有访问权限"))
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// AuthMiddleware JWT认证中间件，用于校验是否登录
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 首先尝试从请求头获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString != "" {
			// 验证Authorization头格式
			parts := strings.Split(tokenString, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			} else {
				utils.HandlerFunc(c, utils.NewBusinessError(utils.ErrCodeAuthTokenInvalid, "认证token格式错误"))
				return
			}
		}

		// 2. 如果请求头中没有，则尝试从URL查询参数中获取
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		// 3. 如果都没有，则返回错误
		if tokenString == "" {
			utils.HandlerFunc(c, utils.NewBusinessError(utils.ErrCodeAuthTokenInvalid, "未提供认证信息"))
			return
		}

		// 解析JWT令牌，使用自定义Claims
		claims, err := parseToken(cfg, tokenString)
		if err != nil {
			utils.HandlerFunc(c, err)
			return
		}

		// 检查Token黑名单
		if rdb := rd.GetClient(); rdb != nil {
			tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenString)))
			blacklistKey := fmt.Sprintf("token:blacklist:%s", tokenHash)
			exists, err := rdb.Exists(c.Request.Context(), blacklistKey).Result()
			if err == nil && exists > 0 {
				utils.HandlerFunc(c, utils.NewBusinessError(utils.ErrCodeTokenRevoked, "认证令牌已失效，请重新登录"))
				c.Abort()
				return
			}
		}

		// 将openID和userID存入上下文
		c.Set("openid", claims.OpenID)
		c.Set("userid", claims.UserID)
		c.Set("user_role", claims.UserRole)
		c.Set("access_token", tokenString)

		c.Next()
	}
}

// 解析JWT令牌
func parseToken(cfg *config.Config, tokenString string) (*CustomClaims, error) {
	secret := []byte(cfg.JWT.JwtSecret)

	// 解析令牌时指定自定义Claims
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, utils.NewBusinessError(utils.ErrCodeTokenTypeInvalid, "无效的签名方法")
		}
		return secret, nil
	})

	if err != nil {
		var validationErr *jwt.ValidationError
		if errors.As(err, &validationErr) {
			if validationErr.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, utils.NewBusinessError(utils.ErrCodeAuthTokenExpired, "认证token已过期")
			}
		}
		return nil, err
	}

	// 验证令牌有效性并转换为自定义Claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		// 检查令牌类型
		if claims.Type != "access" {
			return nil, utils.NewBusinessError(utils.ErrCodeTokenTypeInvalid, "无效的token类型")
		}
		return claims, nil
	}

	return nil, utils.NewSystemError(fmt.Errorf("无效的token"))
}
