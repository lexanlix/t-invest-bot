//nolint:mnd,containedctx
package assembly

import (
	"context"
	"encoding/hex"
	"os"
	"os/signal"
	"syscall"

	"t-api/entity"
	"t-api/internal/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	localConfigPath      = "config/config.yaml"
	botGetUpdatesTimeout = 48 * 60 * 60 // 48 часов
)

type Bootstrap struct {
	logger  *log.Adapter
	config  *entity.Config
	bot     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
	context context.Context
	cancel  context.CancelFunc
	closers []Closer
}

func NewBootstrap() *Bootstrap {
	config, err := entity.ReadConfig(localConfigPath)
	if err != nil {
		panic(errors.WithMessage(err, "read config"))
	}

	logger, err := log.NewFromConfig(*logConfig())
	if err != nil {
		panic(errors.WithMessage(err, "create logger"))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	err = readAesKey(config)
	if err != nil {
		logger.Fatal(ctx, "read key", log.Any("error", err))
	}

	tok := os.Getenv("BOT_TOKEN")

	bot, err := tgbotapi.NewBotAPI(tok)
	if err != nil {
		logger.Fatal(ctx, "create tg bot", log.Any("error", err))
	}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = config.TgBot.Debug

	u := tgbotapi.NewUpdate(0)
	u.Timeout = botGetUpdatesTimeout

	return &Bootstrap{
		logger:  logger,
		config:  config,
		bot:     bot,
		updates: bot.GetUpdatesChan(u),
		context: ctx,
		cancel:  cancel,
		closers: make([]Closer, 0),
	}
}

func (b *Bootstrap) Logger() *log.Adapter {
	return b.logger
}

func (b *Bootstrap) Config() *entity.Config {
	return b.config
}

func (b *Bootstrap) Bot() *tgbotapi.BotAPI {
	return b.bot
}

func (b *Bootstrap) Context() context.Context {
	return b.context
}

func (b *Bootstrap) Start(handleUpdate func(context.Context, tgbotapi.Update)) {
	b.logger.Info(b.context, "tg bot handle started")
	for {
		select {
		// stop looping if ctx is cancelled
		case <-b.context.Done():
			return
		// receive update from channel and then handle it
		case update := <-b.updates:
			handleUpdate(b.context, update)
		}
	}
}

func (b *Bootstrap) Shutdown() {
	b.logger.Info(b.context, "starting shutdown")

	for _, closer := range b.closers {
		closer.Close()
	}

	b.cancel()
	b.logger.Info(context.Background(), "shutdown completed")
}

func (b *Bootstrap) AddCloser(closer Closer) {
	b.closers = append(b.closers, closer)
}

func readAesKey(config *entity.Config) error {
	file, err := os.OpenFile("key.txt", os.O_RDONLY, 0666)
	if err != nil {
		return errors.WithMessage(err, "open key file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return errors.WithMessage(err, "get file stat")
	}

	fileBytes := make([]byte, stat.Size())
	_, err = file.Read(fileBytes)
	if err != nil {
		return errors.WithMessage(err, "read file bytes")
	}

	config.AesKey = make([]byte, hex.DecodedLen(len(fileBytes)))
	_, err = hex.Decode(config.AesKey, fileBytes)
	if err != nil {
		return errors.WithMessage(err, "decoding key")
	}

	return nil
}

func logConfig() *log.Config {
	return &log.Config{
		InitialLevel: -1,
	}
}
