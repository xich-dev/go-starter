//go:build !ut
// +build !ut

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/cloud/sms"
)

func registerAccount(t *testing.T, phone, username, password string) {
	t.Helper()

	te := getTestEngine(t)

	te.POST("/api/v1/auth/code").
		WithJSON(apigen.PostAuthCodeJSONBody{
			Phone: phone,
			Typ:   apigen.Register,
		}).
		Expect().
		Status(202)

	var param = apigen.PostAuthRegisterJSONBody{
		Phone:    phone,
		Code:     sms.FakeCode,
		Username: username,
		Password: password,
	}

	te.POST("/api/v1/auth/register").
		WithJSON(param).
		Expect().
		Status(200)
}

func loginAccount(t *testing.T, _, username, password string) apigen.AuthInfo {
	t.Helper()

	te := getTestEngine(t)

	var authInfo apigen.AuthInfo
	te.POST("/api/v1/auth/login").
		WithJSON(apigen.PostAuthLoginJSONBody{
			UsernameOrPhone: username,
			Password:        password,
		}).
		Expect().
		Status(200).
		JSON().
		Decode(&authInfo)

	te.GET("/api/v1/auth/ping").
		WithHeader("Authorization", "Bearer "+authInfo.Token).
		Expect().
		Status(200)

	var orgInfo apigen.OrgInfoRes
	te.GET("/api/v1/orgs").
		WithHeader("Authorization", "Bearer "+authInfo.Token).
		Expect().
		Status(200).
		JSON().
		Decode(&orgInfo)
	assert.Equal(t, orgInfo.Id, authInfo.OrgID)

	mu.Lock()
	defer mu.Unlock()
	token = authInfo.Token

	return authInfo
}
