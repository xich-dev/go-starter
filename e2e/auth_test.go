//go:build !ut
// +build !ut

package e2e

import (
	"testing"

	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/cloud/sms"
)

func TestChangePassword(t *testing.T) {
	te := getTestEngine(t)

	te.GET("/api/v1/auth/ping").
		Expect().
		Status(401)

	ate := getAuthenticatedTestEngine(t)

	ate.POST("/api/v1/auth/code").
		WithJSON(apigen.PostAuthCodeJSONBody{
			Phone: globalPhone,
			Typ:   apigen.ChangePassword,
		}).
		Expect().
		Status(202)
	ate.POST("/api/v1/auth/change-password").
		WithJSON(apigen.PostAuthChangePasswordJSONBody{
			Phone:       globalPhone,
			NewPassword: globalPassword,
			Code:        sms.FakeCode,
		}).
		Expect().
		Status(200)
}
