//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/xich-dev/go-starter/pkg/apps/server"
	"github.com/xich-dev/go-starter/pkg/cloud/sms"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/controller"
	"github.com/xich-dev/go-starter/pkg/middleware"
	"github.com/xich-dev/go-starter/pkg/model"
	"github.com/xich-dev/go-starter/pkg/service"
)

func InitializeServer() (*server.Server, error) {
	wire.Build(
		config.NewConfig,
		service.NewService,
		controller.NewController,
		model.NewModel,
		server.NewServer,
		middleware.NewMiddleware,
		sms.NewSMSManager,
	)
	return nil, nil
}
