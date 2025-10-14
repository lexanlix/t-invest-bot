package assembly

import (
	"context"
	"encoding/hex"
	"os"

	"t-api/entity"
	"t-api/internal/log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

const (
	localConfigPath      = "conf/config.yaml"
	botGetUpdatesTimeout = 60
)

//nolint:containedctx
type Bootstrap struct {
	ctx     context.Context
	config  *entity.Config
	logger  *log.Adapter
	bot     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel

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

	ctx, cancel := context.WithCancel(context.Background())

	err = readAesKey(config)
	if err != nil {
		logger.Fatal(ctx, "read key", log.Any("error", err))
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		logger.Fatal(ctx, "create tg bot", log.Any("error", err))
	}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = config.TgBot.Debug

	u := tgbotapi.NewUpdate(0)
	u.Timeout = botGetUpdatesTimeout

	return &Bootstrap{
		ctx:     ctx,
		config:  config,
		logger:  logger,
		bot:     bot,
		updates: bot.GetUpdatesChan(u),
		cancel:  cancel,
		closers: make([]Closer, 0),
	}
}

func (b *Bootstrap) Context() context.Context {
	return b.ctx
}

func (b *Bootstrap) Config() *entity.Config {
	return b.config
}

func (b *Bootstrap) Logger() *log.Adapter {
	return b.logger
}

func (b *Bootstrap) Bot() *tgbotapi.BotAPI {
	return b.bot
}

func (b *Bootstrap) Start(handleUpdate func(context.Context, tgbotapi.Update)) {
	b.logger.Info(b.ctx, "tg bot handle started")
	for {
		select {
		// stop looping if ctx is cancelled
		case <-b.ctx.Done():
			return
		// receive update from channel and then handle it
		case update := <-b.updates:
			handleUpdate(b.ctx, update)
		}
	}
}

func (b *Bootstrap) Shutdown() {
	for _, closer := range b.closers {
		closer.Close()
	}

	b.cancel()
}

func (b *Bootstrap) AddClosers(closers ...Closer) {
	b.closers = append(b.closers, closers...)
}

func readAesKey(config *entity.Config) error {
	keyStr := os.Getenv("AES_KEY")
	if len(keyStr) == 0 {
		return errors.New("no AES_KEY env var")
	}

	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return errors.WithMessage(err, "decoding key")
	}

	config.AesKey = key
	return nil
}

func logConfig() *log.Config {
	return &log.Config{
		InitialLevel: -1,
	}
}
