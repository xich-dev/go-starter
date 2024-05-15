package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/cloud/sms"
	"github.com/xich-dev/go-starter/pkg/model"
	"github.com/xich-dev/go-starter/pkg/model/querier"
	"github.com/xich-dev/go-starter/pkg/utils"
)

func TestCreateCode_exist_no_expire(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// params
	var (
		phone = "18088805143"
		code  = "12345"
		typ   = "Register"
	)

	// mocking modules
	var (
		mockModel = model.NewMockModelInterface(ctrl)
		mockSMS   = sms.NewMockSMSManagerInterface(ctrl)
	)

	// mock function calls
	mockModel.
		EXPECT().
		GetPhoneCode(gomock.Any(), querier.GetPhoneCodeParams{
			Phone: phone,
			Typ:   typ,
		}).
		Return(&querier.PhoneCode{
			Phone:     phone,
			Code:      code,
			Typ:       typ,
			Used:      false,
			ExpiredAt: time.Now().Add(5 * time.Minute),
			UpdatedAt: time.Now(),
		}, nil)
	// test case
	svc := &Service{
		m:          mockModel,
		smsManager: mockSMS,
		now:        time.Now,
	}
	err := svc.CreateCode(context.Background(), apigen.PostAuthCodeJSONBody{
		Phone: phone,
		Typ:   apigen.PostAuthCodeJSONBodyTyp(typ),
	})

	assert.Equal(t, ErrCodeNotExpired, err)
}

func TestCreateCode_exist_expire(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// params
	var (
		phone       = "18088805143"
		code        = "9527"
		typ         = "Register"
		nowTime     = time.Now()
		newNxpireAt = nowTime.Add(ExpireDuration)
	)

	// mocking modules
	var (
		mockModel = model.NewMockModelInterface(ctrl)
		mockSMS   = sms.NewMockSMSManagerInterface(ctrl)
	)

	// mock function calls
	mockModel.
		EXPECT().
		GetPhoneCode(gomock.Any(), querier.GetPhoneCodeParams{
			Phone: phone,
			Typ:   typ,
		}).
		Return(&querier.PhoneCode{
			Phone:     phone,
			Code:      code,
			Typ:       typ,
			Used:      false,
			ExpiredAt: nowTime.Add(-ExpireDuration),
			UpdatedAt: nowTime,
		}, nil)

	mockModel.
		EXPECT().
		UpsertPhoneCode(gomock.Any(), querier.UpsertPhoneCodeParams{
			Phone:     phone,
			Typ:       typ,
			Code:      code,
			ExpiredAt: newNxpireAt,
		}).
		Return(&querier.PhoneCode{
			Phone:     phone,
			Typ:       typ,
			Code:      code,
			ExpiredAt: newNxpireAt,
			Used:      false,
			UpdatedAt: nowTime,
		}, nil)

	mockSMS.
		EXPECT().
		GenerateCode().
		Return(code)

	mockSMS.
		EXPECT().
		SendCode(phone, code).
		Return(nil)

	// test case
	svc := &Service{
		m:          mockModel,
		smsManager: mockSMS,
		now: func() time.Time {
			return nowTime
		},
	}
	err := svc.CreateCode(context.Background(), apigen.PostAuthCodeJSONBody{
		Phone: phone,
		Typ:   apigen.PostAuthCodeJSONBodyTyp(typ),
	})

	assert.NoError(t, err)
}

func TestCreateCode_not_exist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// params
	var (
		phone       = "18088805143"
		code        = "9527"
		typ         = "Register"
		nowTime     = time.Now()
		newNxpireAt = nowTime.Add(ExpireDuration)
	)

	// mocking modules
	var (
		mockModel = model.NewMockModelInterface(ctrl)
		mockSMS   = sms.NewMockSMSManagerInterface(ctrl)
	)

	// mock function calls
	mockModel.
		EXPECT().
		GetPhoneCode(gomock.Any(), querier.GetPhoneCodeParams{
			Phone: phone,
			Typ:   typ,
		}).
		Return(nil, pgx.ErrNoRows)

	mockModel.
		EXPECT().
		UpsertPhoneCode(gomock.Any(), querier.UpsertPhoneCodeParams{
			Phone:     phone,
			Typ:       typ,
			Code:      code,
			ExpiredAt: newNxpireAt,
		}).
		Return(&querier.PhoneCode{
			Phone:     phone,
			Typ:       typ,
			Code:      code,
			ExpiredAt: newNxpireAt,
			Used:      false,
			UpdatedAt: nowTime,
		}, nil)

	mockSMS.
		EXPECT().
		GenerateCode().
		Return(code)

	mockSMS.
		EXPECT().
		SendCode(phone, code).
		Return(nil)

	// test case
	svc := &Service{
		m:          mockModel,
		smsManager: mockSMS,
		now: func() time.Time {
			return nowTime
		},
	}
	err := svc.CreateCode(context.Background(), apigen.PostAuthCodeJSONBody{
		Phone: phone,
		Typ:   apigen.PostAuthCodeJSONBodyTyp(typ),
	})

	assert.NoError(t, err)
}

func TestCreateUserWithNewOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		ctx      = context.Background()
		phone    = "18088805143"
		username = "mike"
		password = "password"
		userID   = uuid.Must(uuid.NewRandom())
		orgID    = uuid.Must(uuid.NewRandom())
	)
	salt, hashedPassword, err := utils.GenerateHashAndSalt(password)
	require.NoError(t, err)

	mockModel := model.NewExtendedMockModelInterface(ctrl)

	mockModel.
		EXPECT().
		IsUsernameExist(ctx, username).
		Return(false, nil)

	mockModel.
		EXPECT().
		IsPhoneExist(ctx, phone).
		Return(false, nil)

	mockModel.
		EXPECT().
		CreateUser(ctx, querier.CreateUserParams{
			Name:         username,
			Phone:        phone,
			PasswordHash: hashedPassword,
			PasswordSalt: salt,
			OrgID:        orgID,
		}).
		Return(&querier.User{
			ID: userID,
		}, nil)

	mockModel.
		EXPECT().
		CreateOrg(ctx, "mike的小组").
		Return(&querier.Org{
			ID:   orgID,
			Name: "mike的小组",
		}, nil)

	mockModel.
		EXPECT().
		UpdateOrgOwnerID(ctx, querier.UpdateOrgOwnerIDParams{
			OwnerID: uuid.NullUUID{Valid: true, UUID: userID},
			ID:      orgID,
		})

	svc := &Service{
		m: mockModel,
		generateHashAndSalt: func(password string) (string, string, error) {
			return salt, hashedPassword, nil
		},
	}

	err = svc.CreateUserWithNewOrg(ctx, apigen.PostAuthRegisterJSONBody{
		Username: username,
		Phone:    phone,
		Password: password,
	})

	assert.NoError(t, err)
}

func TestVerifyCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		phone = "18088805143"
		code  = "9527"
		typ   = apigen.Register
	)

	mockModel := model.NewExtendedMockModelInterface(ctrl)

	mockModel.
		EXPECT().
		GetPhoneCode(gomock.Any(), querier.GetPhoneCodeParams{
			Phone: phone,
			Typ:   string(typ),
		}).
		Return(&querier.PhoneCode{
			Phone:     phone,
			Code:      code,
			Typ:       string(typ),
			Used:      false,
			ExpiredAt: time.Now().Add(5 * time.Minute),
		}, nil)

	mockModel.
		EXPECT().
		MarkPhoneCodeUsed(gomock.Any(), querier.MarkPhoneCodeUsedParams{
			Phone: phone,
			Typ:   string(typ),
		}).
		Return(nil)

	svc := &Service{
		m:   mockModel,
		now: time.Now,
	}

	err := svc.VerifyCode(context.Background(), phone, typ, code)
	assert.NoError(t, err)
}

func TestVerifyCode_exceptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		phone = "18088805143"
		code  = "9527"
		typ   = apigen.Register
	)

	testCases := []struct {
		code        string
		phoneCode   *querier.PhoneCode
		expectedErr error
	}{
		{
			code: "wrong-code",
			phoneCode: &querier.PhoneCode{
				Phone:     phone,
				Code:      code,
				Typ:       string(typ),
				Used:      false,
				ExpiredAt: time.Now().Add(5 * time.Minute),
			},
			expectedErr: ErrCodeInvalid,
		},
		{
			code: code,
			phoneCode: &querier.PhoneCode{
				Phone:     phone,
				Code:      code,
				Typ:       string(typ),
				Used:      false,
				ExpiredAt: time.Now().Add(-5 * time.Minute),
			},
			expectedErr: ErrCodeExpire,
		},
		// {
		// 	code: code,
		// 	phoneCode: &querier.PhoneCode{
		// 		Phone:     phone,
		// 		Code:      code,
		// 		Typ:       string(typ),
		// 		Used:      true,
		// 		ExpiredAt: time.Now().Add(5 * time.Minute),
		// 	},
		// 	expectedErr: ErrCodeUsed,
		// },
	}

	for _, testCase := range testCases {
		mockModel := model.NewExtendedMockModelInterface(ctrl)
		mockModel.
			EXPECT().
			GetPhoneCode(gomock.Any(), querier.GetPhoneCodeParams{
				Phone: phone,
				Typ:   string(typ),
			}).
			Return(testCase.phoneCode, nil)

		svc := &Service{
			m:   mockModel,
			now: time.Now,
		}

		err := svc.VerifyCode(context.Background(), phone, typ, testCase.code)
		assert.Equal(t, testCase.expectedErr, err)
	}
}

func TestVerifyLoginInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var (
		phone    = "15767550998"
		password = "password"
	)
	salt, hashedPassword, err := utils.GenerateHashAndSalt(password)
	require.NoError(t, err)

	testCases := []struct {
		phone       string
		password    string
		userInfo    *querier.User
		expectedErr error
	}{
		{
			phone:    phone,
			password: password,
			userInfo: &querier.User{
				PasswordHash: hashedPassword,
				PasswordSalt: salt,
			},
			expectedErr: nil,
		},
		{
			phone:       "wrong-phone",
			password:    password,
			userInfo:    nil,
			expectedErr: ErrUsernameOrPhoneNotFound,
		},
		{
			phone:    phone,
			password: "wrong-password",
			userInfo: &querier.User{
				PasswordHash: hashedPassword,
				PasswordSalt: salt,
			},
			expectedErr: ErrIncorrectPassword,
		},
		{
			phone:    phone,
			password: password,
			userInfo: &querier.User{
				PasswordHash: hashedPassword,
				PasswordSalt: salt,
				DeletedAt:    &time.Time{},
			},
			expectedErr: ErrDeletedUser,
		},
	}
	for _, testCase := range testCases {
		mockModel := model.NewExtendedMockModelInterface(ctrl)

		if testCase.userInfo != nil {
			mockModel.
				EXPECT().
				GetUser(gomock.Any(), testCase.phone).
				Return(testCase.userInfo, nil)
			if testCase.expectedErr == nil {
				mockModel.
					EXPECT().
					GetUserAccessRuleNames(gomock.Any(), testCase.userInfo.ID).
					Return([]string{
						"rule1",
					}, nil)
			}
		} else {
			mockModel.
				EXPECT().
				GetUser(gomock.Any(), testCase.phone).
				Return(nil, ErrUsernameOrPhoneNotFound)
		}

		svc := &Service{
			m:   mockModel,
			now: time.Now,
		}

		_, rules, err := svc.VerifyLoginInfo(context.Background(), apigen.PostAuthLoginJSONBody{
			Password:        testCase.password,
			UsernameOrPhone: testCase.phone,
		})
		if testCase.expectedErr != nil {
			assert.True(t, errors.Is(err, testCase.expectedErr))
		} else {
			assert.NoError(t, err)
			assert.Equal(t, []string{"rule1"}, rules)
		}
	}
}

func TestChangePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var (
		ctx         = context.Background()
		phone       = "18088805143"
		newPassword = "password"
	)
	salt, hashedPassword, err := utils.GenerateHashAndSalt(newPassword)
	require.NoError(t, err)

	mockModel := model.NewExtendedMockModelInterface(ctrl)
	mockModel.EXPECT().UpdateUserPasswordByPhone(gomock.Any(), querier.UpdateUserPasswordByPhoneParams{
		Phone:        phone,
		PasswordHash: hashedPassword,
		PasswordSalt: salt,
	}).Return(nil)
	svc := &Service{
		m: mockModel,
		generateHashAndSalt: func(password string) (string, string, error) {
			return salt, hashedPassword, nil
		},
	}
	err = svc.ChangePassword(ctx, apigen.PostAuthChangePasswordJSONBody{
		Phone:       phone,
		NewPassword: newPassword,
	})
	assert.NoError(t, err)
}

func TestGetOrgsInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	orgId := uuid.Must(uuid.NewRandom())

	testCases := []struct {
		orgId       uuid.UUID
		orgInfo     *querier.Org
		expectedErr error
	}{
		{
			orgId: orgId,
			orgInfo: &querier.Org{
				ID:   orgId,
				Name: "orgName",
			},
			expectedErr: nil,
		},
		{
			orgId:       uuid.Must(uuid.NewRandom()),
			orgInfo:     nil,
			expectedErr: ErrOrgNotFound,
		},
	}
	mockModel := model.NewExtendedMockModelInterface(ctrl)
	svc := &Service{
		m: mockModel,
	}
	for _, testCase := range testCases {
		if testCase.orgInfo != nil {
			mockModel.
				EXPECT().
				GetOrgInfoByOrgId(gomock.Any(), testCase.orgId).
				Return(testCase.orgInfo, nil)
		} else {
			mockModel.
				EXPECT().
				GetOrgInfoByOrgId(gomock.Any(), testCase.orgId).
				Return(nil, pgx.ErrNoRows)
		}
		org, err := svc.GetOrgInfoByOrgId(context.Background(), testCase.orgId)
		if testCase.expectedErr != nil {
			assert.True(t, errors.Is(err, testCase.expectedErr))
		} else {
			assert.NoError(t, err)
			assert.Equal(t, org.Id, testCase.orgId)
		}
	}
}
