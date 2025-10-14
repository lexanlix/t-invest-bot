package main

import (
	"t-api/assembly"
	"t-api/internal/shutdown"
)

func main() {
	boot := assembly.NewBootstrap()
	logger := boot.Logger()

	shutdown.On(func() {
		logger.Info(boot.Context(), "starting shutdown")
		boot.Shutdown()
		logger.Info(boot.Context(), "shutdown completed")
	})

	logger.Info(boot.Context(), "creating telegram service")
	tg, err := assembly.NewTelegram(boot.Context(), logger, *boot.Config(), boot.Bot())
	if err != nil {
		logger.Fatal(boot.Context(), err)
	}

	boot.AddClosers(tg)
	boot.Start(tg.HandleUpdate)
}
