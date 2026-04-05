package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/nk87rus/stenographer/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	appInstance, err := app.Init(ctx)
	if err != nil {
		log.Fatalf("ошибка при инициализации приложения: %v", err)
	}

	appInstance.Run(ctx)

	<-ctx.Done()
	// shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer shutdownCancel()

	log.Println("Graceful shutdown выполнен успешно")
}
