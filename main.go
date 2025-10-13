package main

import "t-api/assembly"

func main() {
	boot := assembly.NewBootstrap()
	defer boot.Shutdown()

	logger := boot.Logger()

	logger.Info(boot.Context(), "creating telegram service")
	tg, err := assembly.NewTelegram(boot.Context(), logger, *boot.Config(), boot.Bot())
	if err != nil {
		logger.Fatal(boot.Context(), err)
	}

	boot.AddCloser(tg)
	boot.Start(tg.HandleUpdate)
}
