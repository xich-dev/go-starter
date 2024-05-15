package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/cloud/sms"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/model"
	"github.com/xich-dev/go-starter/pkg/model/querier"
	"github.com/xich-dev/go-starter/pkg/utils"
)

type (
	TradeType   string
	TradeStatus string
	DdWorkEvent string
)

var (
	ErrCodeUsed                = errors.New("code used")
	ErrCodeNotFound            = errors.New("code not found")
	ErrCodeNotExpired          = errors.New("code is not expired")
	ErrCodeExpire              = errors.New("code expired")
	ErrCodeInvalid             = errors.New("invalid code")
	ErrUsernameAlreadyExist    = errors.New("用户名已注册")
	ErrPhoneAlreadyExist       = errors.New("手机号已注册")
	ErrUsernameOrPhoneNotFound = errors.New("用户名或手机号不存在")
	ErrDeletedUser             = errors.New("用户名或手机号不存在")
	ErrIncorrectPassword       = errors.New("密码错误")
	ErrInvalidParams           = errors.New("参数错误")

	//trade
	ErrInsufficientBalance = errors.New("余额不足，请充值")

	//org
	ErrOrgNotFound = errors.New("组织不存在")
)

const (
	ExpireDuration    = 2 * time.Minute
	DefaultMaxRetries = 3
)

type ServiceInterface interface {
	// orgs

	CreateCode(ctx context.Context, param apigen.PostAuthCodeJSONBody) error

	CreateUserWithNewOrg(ctx context.Context, param apigen.PostAuthRegisterJSONBody) error

	VerifyCode(ctx context.Context, phone string, typ apigen.PostAuthCodeJSONBodyTyp, code string) error

	VerifyLoginInfo(ctx context.Context, param apigen.PostAuthLoginJSONBody) (*querier.User, []string, error)

	ChangePassword(ctx context.Context, param apigen.PostAuthChangePasswordJSONBody) error

	GetOrgInfoByOrgId(ctx context.Context, ID uuid.UUID) (*apigen.OrgInfoRes, error)

	// for Testing
	AddUserAccessRuleByUsername(ctx context.Context, username string, ruleNames ...string) error
}

type Service struct {
	m          model.ModelInterface
	smsManager sms.SMSManagerInterface

	now                 func() time.Time
	generateHashAndSalt func(password string) (string, string, error)
}

func NewService(cfg *config.Config, m model.ModelInterface, smsManager sms.SMSManagerInterface) ServiceInterface {
	return &Service{
		m:                   m,
		smsManager:          smsManager,
		now:                 time.Now,
		generateHashAndSalt: utils.GenerateHashAndSalt,
	}
}

// CreateCode creates a new code for the phone number.
// if the code exists and not expired, return ErrNotExpired
// if the code does not exist or it is expired, create a new code and send it to the phone number
func (s *Service) CreateCode(ctx context.Context, param apigen.PostAuthCodeJSONBody) error {
	code, err := s.m.GetPhoneCode(ctx, querier.GetPhoneCodeParams{
		Phone: param.Phone,
		Typ:   string(param.Typ),
	})

	// check if the code exists
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return errors.Wrap(err, "failed to get phone code")
		}
	}
	// check if the code is expired
	if code != nil && code.ExpiredAt.After(s.now()) {
		return ErrCodeNotExpired
	}

	newCode := s.smsManager.GenerateCode()

	_, err = s.m.UpsertPhoneCode(ctx, querier.UpsertPhoneCodeParams{
		Phone:     param.Phone,
		Code:      newCode,
		Typ:       string(param.Typ),
		ExpiredAt: s.now().Add(ExpireDuration),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create phone code")
	}

	if err := s.smsManager.SendCode(param.Phone, newCode); err != nil {
		return errors.Wrap(err, "failed to send vcode")
	}
	return nil
}

func (s *Service) VerifyCode(ctx context.Context, phone string, typ apigen.PostAuthCodeJSONBodyTyp, code string) error {
	return s.m.RunTransaction(ctx, func(model model.ModelInterface) error {
		phoneCode, err := s.m.GetPhoneCode(ctx, querier.GetPhoneCodeParams{
			Phone: phone,
			Typ:   string(typ),
		})
		// if phoneCode.Used {
		// 	return ErrCodeUsed
		// }
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCodeNotFound
			}
			return errors.Wrap(err, "failed to get phone code")
		}
		if phoneCode.ExpiredAt.Before(s.now()) {
			return ErrCodeExpire
		}
		if phoneCode.Code != code {
			return ErrCodeInvalid
		}
		if err := s.m.MarkPhoneCodeUsed(ctx, querier.MarkPhoneCodeUsedParams{
			Phone: phone,
			Typ:   string(typ),
		}); err != nil {
			return errors.Wrap(err, "failed to mark phone code used")
		}
		return nil
	})
}

func CentsToCoins[T int | int32 | uint32 | uint64 | int64](cents T) string {
	left := fmt.Sprintf("%d", cents/10)
	rightNum := cents % 10
	if rightNum == 0 {
		return left
	}
	return fmt.Sprintf("%s.%d", left, rightNum)
}

func (s *Service) CreateUserWithNewOrg(ctx context.Context, param apigen.PostAuthRegisterJSONBody) error {
	salt, hashedPassword, err := s.generateHashAndSalt(param.Password)
	if err != nil {
		return errors.Wrap(err, "failed to generate hash and salt")
	}

	return s.m.RunTransaction(ctx, func(model model.ModelInterface) error {
		exist, err := model.IsUsernameExist(ctx, param.Username)
		if err != nil {
			return errors.Wrap(err, "failed to check username exist")
		}
		if exist {
			return ErrUsernameAlreadyExist
		}

		exist, err = model.IsPhoneExist(ctx, param.Phone)
		if err != nil {
			return errors.Wrap(err, "failed to check phone exist")
		}
		if exist {
			return ErrPhoneAlreadyExist
		}

		// create default org for users
		org, err := model.CreateOrg(ctx, fmt.Sprintf("%s的小组", param.Username))
		if err != nil {
			return errors.Wrap(err, "failed to create org")
		}

		// create user
		user, err := model.CreateUser(ctx, querier.CreateUserParams{
			OrgID:        org.ID,
			Name:         param.Username,
			Phone:        param.Phone,
			PasswordHash: hashedPassword,
			PasswordSalt: salt,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create user")
		}

		// update org owner ID
		if err := model.UpdateOrgOwnerID(ctx, querier.UpdateOrgOwnerIDParams{
			OwnerID: uuid.NullUUID{Valid: true, UUID: user.ID},
			ID:      org.ID,
		}); err != nil {
			return errors.Wrap(err, "failed to update org owner id")
		}

		return nil
	})
}

func (s *Service) VerifyLoginInfo(ctx context.Context, param apigen.PostAuthLoginJSONBody) (*querier.User, []string, error) {
	user, err := s.m.GetUser(ctx, param.UsernameOrPhone)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, errors.Wrap(err, "failed to get user")
		}
		return nil, nil, ErrUsernameOrPhoneNotFound
	}
	if user.DeletedAt != nil {
		return nil, nil, ErrDeletedUser
	}
	inputHashPassword, err := utils.HashPassword(param.Password, user.PasswordSalt)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to hash password")
	}
	if inputHashPassword != user.PasswordHash {
		return nil, nil, ErrIncorrectPassword
	}
	rules, err := s.m.GetUserAccessRuleNames(ctx, user.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get user access rules")
	}
	return user, rules, nil
}

func (s *Service) ChangePassword(ctx context.Context, param apigen.PostAuthChangePasswordJSONBody) error {
	salt, hashedPassword, err := s.generateHashAndSalt(param.NewPassword)
	if err != nil {
		return errors.Wrap(err, "failed to generate hash and salt")
	}
	if err := s.m.UpdateUserPasswordByPhone(ctx, querier.UpdateUserPasswordByPhoneParams{
		Phone:        param.Phone,
		PasswordHash: hashedPassword,
		PasswordSalt: salt,
	}); err != nil {
		return errors.Wrap(err, "failed to reset password")
	}
	return nil
}

func (s *Service) AddUserAccessRuleByUsername(ctx context.Context, username string, ruleNames ...string) error {
	for _, ruleName := range ruleNames {
		r, err := s.m.GetAccessRule(ctx, ruleName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.Errorf("no such access rule: %s", ruleName)
			} else {
				return errors.Wrap(err, "failed to get access rule")
			}
		}

		if err := s.m.AddUserAccessRule(ctx, querier.AddUserAccessRuleParams{
			Name:   username,
			RuleID: r.ID,
		}); err != nil {
			return errors.Wrapf(err, "failed to add access rule %s to user %s", ruleName, username)
		}
	}
	return nil
}

func (s *Service) GetOrgInfoByOrgId(ctx context.Context, ID uuid.UUID) (*apigen.OrgInfoRes, error) {
	orgInfo, err := s.m.GetOrgInfoByOrgId(ctx, ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		} else {
			return nil, errors.Wrap(err, "failed to get org info by org id")
		}
	}

	var ownerId *uuid.UUID
	if orgInfo.OwnerID.Valid {
		ownerId = &orgInfo.OwnerID.UUID
	}

	return &apigen.OrgInfoRes{
		Id:      orgInfo.ID,
		Name:    orgInfo.Name,
		OwnerId: ownerId,
	}, nil
}
