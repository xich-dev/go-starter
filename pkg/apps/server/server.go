package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/config"
	"github.com/xich-dev/go-starter/pkg/controller"
	"github.com/xich-dev/go-starter/pkg/middleware"
)

type Server struct {
	app        *fiber.App
	port       int
	middleware *middleware.Middleware
	controller *controller.Controller
}

func NewServer(cfg *config.Config, c *controller.Controller, middleware *middleware.Middleware) *Server {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    50 * 1024 * 1024, // 50MB
	})

	s := &Server{
		app:        app,
		port:       cfg.Port,
		middleware: middleware,
		controller: c,
	}

	s.registerMiddleware()

	apigen.RegisterHandlersWithOptions(s.app, s.controller, apigen.FiberServerOptions{
		BaseURL:     "/api/v1",
		Middlewares: []apigen.MiddlewareFunc{},
	})

	return s
}

func (s *Server) GetController() *controller.Controller {
	return s.controller
}

func (s *Server) registerMiddleware() {
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	s.app.Use(cors.New(cors.Config{}))
	s.app.Use(requestid.New())
	s.app.Use(middleware.NewLogger())

	s.app.Get("/api/v1/auth/ping", s.middleware.Auth())
	s.app.Get("/api/v1/orgs", s.middleware.Auth())

}

func (s *Server) Listen() error {
	return s.app.Listen(fmt.Sprintf(":%d", s.port))
}

func (s *Server) GetApp() *fiber.App {
	return s.app
}
