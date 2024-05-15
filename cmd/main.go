package main

import (
	"github.com/xich-dev/go-starter/pkg/logger"
	"github.com/xich-dev/go-starter/wire"
	"go.uber.org/zap"
)

var log = logger.NewLogAgent("main")

func main() {
	s, err := wire.InitializeServer()
	if err != nil {
		log.Error("failed to initialize server", zap.Error(err))
		panic(err)
	}

	if err := s.Listen(); err != nil {
		log.Error("exit with error", zap.Error(err))
	}

	log.Info("bye.")
}
