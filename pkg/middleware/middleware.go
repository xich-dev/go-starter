package middleware

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/model/querier"
	"github.com/xich-dev/go-starter/pkg/utils"
)

const (
	UserContextKey     = "user"
	jwtTokenContextKey = "jwt_token"
)

var (
	ErrUserIdentityNotExist = errors.New("user identity not exists")
)

type User struct {
	Id          uuid.UUID
	OrgID       uuid.UUID
	AccessRules map[string]struct{}
}

type Middleware struct {
	jwtMiddleware func(*fiber.Ctx) error
	jwtSecret     []byte
}

func NewMiddleware(cfg *config.Config) (*Middleware, error) {
	if len(cfg.Jwt.Secret) == 0 {
		return nil, errors.New("jwt secret is empty")
	}

	return &Middleware{
		jwtMiddleware: jwtware.New(jwtware.Config{
			SigningKey:     jwtware.SigningKey{Key: []byte(cfg.Jwt.Secret)},
			ContextKey:     jwtTokenContextKey,
			SuccessHandler: func(c *fiber.Ctx) error { return nil },
		}),
		jwtSecret: []byte(cfg.Jwt.Secret),
	}, nil
}

func (m *Middleware) CreateToken(user *querier.User, rules []string) (string, error) {
	claims := m.CreateClaims(user, rules)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.jwtSecret))
}

func (m *Middleware) Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := m.jwtMiddleware(c); err != nil {
			return c.Status(403).SendString(err.Error())
		}
		user, err := parseClaims(c)
		if err != nil {
			return c.Status(401).SendString(err.Error())
		}
		c.Locals(UserContextKey, user)
		return c.Next()
	}
}

func (m *Middleware) CheckRules(rules []string, rejectRules []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := GetUser(c)
		if err != nil {
			return c.Status(403).SendString(err.Error())
		}
		for _, rule := range rules {
			if _, ok := user.AccessRules[rule]; !ok {
				return c.Status(403).SendString(fmt.Sprintf("没有权限，需要访问规则%s", rule))
			}
		}
		for _, rule := range rejectRules {
			if _, ok := user.AccessRules[rule]; ok {
				return c.Status(403).SendString(fmt.Sprintf("没有权限，因为带有访问规则%s", rule))
			}
		}
		return c.Next()
	}
}

func GetUser(c *fiber.Ctx) (*User, error) {
	user, ok := c.Locals(UserContextKey).(*User)
	if !ok {
		return nil, ErrUserIdentityNotExist
	}
	return user, nil
}

func (m *Middleware) CreateClaims(user *querier.User, accessRules []string) jwt.MapClaims {
	ruleMap := make(map[string]struct{})
	for _, rule := range accessRules {
		ruleMap[rule] = struct{}{}
	}

	return jwt.MapClaims{
		"user": &User{
			Id:          user.ID,
			OrgID:       user.OrgID,
			AccessRules: ruleMap,
		},
		"exp": time.Now().Add(12 * time.Hour).Unix(),
	}
}

func parseClaims(c *fiber.Ctx) (*User, error) {
	user, ok := c.Locals(jwtTokenContextKey).(*jwt.Token)
	if !ok {
		return nil, errors.New("unexpected error when parsing claims: user field is not jwt token")
	}
	claims, ok := user.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("unexpected error when parsing claims: claims is not jwt.MapClaims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("failed to parse exp")
	}
	if time.Since(time.Unix(int64(exp), 0)) > 0 {
		return nil, errors.New("token is expired")
	}
	var u User
	if err := utils.JSONConvert(claims["user"], &u); err != nil {
		return nil, errors.Wrapf(err, "failed to parse user from claims: %s", utils.TryMarshal(claims["user"]))
	}
	return &u, nil
}
