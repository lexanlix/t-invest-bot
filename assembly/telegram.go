package assembly

import (
	"context"

	"t-api/entity"
	"t-api/internal/log"
	"t-api/repository"
	"t-api/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"
)

type Telegram struct {
	logger     log.Logger
	bot        *tgbotapi.BotAPI
	config     entity.Config
	usersCache *Cache
}

func NewTelegram(ctx context.Context, logger log.Logger, config entity.Config, bot *tgbotapi.BotAPI) (*Telegram, error) {
	cache, err := NewCache(config.Cache.Filepath, config.AesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "create cache")
	}

	t := &Telegram{
		logger:     logger,
		bot:        bot,
		config:     config,
		usersCache: cache,
	}

	for _, user := range t.usersCache.GetAll() {
		if user.data.State == entity.StateSelectedAccount {
			err = t.upUserService(ctx, user)
			if err != nil {
				return nil, errors.WithMessagef(err, "up user service, user '%s", user.data.Username)
			}
		}
	}

	return t, nil
}

func (t Telegram) Close() error {
	err := t.usersCache.SaveToFile()
	if err != nil {
		t.logger.Error(context.Background(), "save telegram cache", log.Any("error", err))
	}

	return nil
}

func (t Telegram) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		t.handleMessage(ctx, update.Message)
		break

	case update.CallbackQuery != nil:
		t.handleButton(ctx, update.CallbackQuery)
		break
	}
}

func (t Telegram) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	from := message.From
	text := message.Text

	if from == nil {
		return
	}

	t.logger.Info(ctx, "handle message", log.String("from", from.FirstName), log.String("text", text))

	user, found := t.usersCache.Get(from.ID)
	if found {
		t.navigateUser(ctx, user, text)
		return
	}

	err := t.createUser(from, text)
	if err != nil {
		t.logger.Error(ctx, "create user", log.Any("error", err))
	}
}

func (t Telegram) handleButton(ctx context.Context, query *tgbotapi.CallbackQuery) {
	from := query.From
	text := query.Data

	if from == nil {
		return
	}

	t.logger.Info(ctx, "handle button", log.String("from", from.FirstName), log.String("data", text))

	user, found := t.usersCache.Get(from.ID)
	if found {
		t.navigateUser(ctx, user, text)
		return
	}

	err := t.createUser(from, text)
	if err != nil {
		t.logger.Error(ctx, "create user", log.Any("error", err))
	}
}

func (t Telegram) navigateUser(ctx context.Context, user User, msgText string) {
	switch user.data.State {
	case entity.StateStart:
		return
	case entity.StateUserCreated:
		err := t.getInvestToken(ctx, user, msgText)
		if err != nil {
			t.logger.Error(ctx, "get invest token", log.Any("error", err))
		}
		user.data.State = entity.StateGotInvestToken
		return
	case entity.StateGotInvestToken:
		err := t.selectAccount(ctx, user, msgText)
		if err != nil {
			t.logger.Error(ctx, "select account", log.Any("error", err))
		}
		user.data.State = entity.StateSelectedAccount
		return
	case entity.StateSelectedAccount:
		t.handleCommand(user, msgText)
	default:
		return
	}
}

func (t Telegram) handleCommand(user User, command string) {
	switch command {
	case entity.StopButtonCommand:
		user.service.PauseTracing()

		keyboardMsg := tgbotapi.NewMessage(user.data.ChatId, "Отслеживание приостановлено")
		keyboardMsg.ReplyMarkup = createContinueButtonMarkup()
		_, err := t.bot.Send(keyboardMsg)
		if err != nil {
			t.logger.Error(context.Background(), "send keyboard message", log.Any("error", err))
		}
	case entity.ContinueButtonCommand:
		user.service.StartTracing()

		keyboardMsg := tgbotapi.NewMessage(user.data.ChatId, entity.StartTracingMsg+" "+user.data.Invest.TracingAccountName)
		keyboardMsg.ReplyMarkup = createStopButtonMarkup()
		_, err := t.bot.Send(keyboardMsg)
		if err != nil {
			t.logger.Error(context.Background(), "send keyboard message", log.Any("error", err))
		}
	case entity.AccountsButtonCommand:
		user.data.State = entity.StateGotInvestToken
		defer t.usersCache.Update(user.data.ChatId, user)

		accounts, err := user.service.GetAccounts()
		if err != nil {
			user.data.State = entity.StateUserCreated
			t.logger.Error(context.Background(), "get accounts", log.Any("error", err))
			return
		}

		accNameById := make(map[string]string)
		for _, acc := range accounts {
			accNameById[acc.GetId()] = acc.GetName()
		}
		user.data.Invest.AccountNameById = accNameById

		accountsMsg := tgbotapi.NewMessage(user.data.ChatId,
			"Отслеживание счета "+user.data.Invest.TracingAccountName+" приостановлено. Выберите счет")
		accountsMsg.ReplyMarkup = createAccountsMarkup(accounts)

		_, err = t.bot.Send(accountsMsg)
		if err != nil {
			t.logger.Error(context.Background(), "send accounts message", log.Any("error", err))
		}
	}
}

func (t Telegram) createUser(userData *tgbotapi.User, command string) error {
	if command != entity.StartCommand {
		return nil
	}

	user := User{
		data: &entity.User{
			ChatId:   userData.ID,
			Username: userData.UserName,
			State:    entity.StateUserCreated,
		},
	}

	t.usersCache.Add(user.data.ChatId, user)

	_, err := t.bot.Send(tgbotapi.NewMessage(user.data.ChatId, entity.GiveTokenMsg))
	if err != nil {
		return errors.WithMessagef(err, "send telegram message '%s'", entity.GiveTokenMsg)
	}

	return nil
}

func (t Telegram) getInvestToken(ctx context.Context, user User, token string) error {
	user.data.Invest = entity.Invest{
		Config: t.config.Invest,
	}
	user.data.Invest.Token = token

	investRepo, err := repository.NewInvest(ctx, user.data.Invest.Config)
	if err != nil {
		return errors.WithMessage(err, "create invest repository")
	}

	accounts, err := investRepo.GetUserAccounts()
	if err != nil {
		return errors.WithMessage(err, "get user accounts")
	}

	accNameById := make(map[string]string)
	for _, acc := range accounts {
		accNameById[acc.GetId()] = acc.GetName()
	}
	user.data.Invest.AccountNameById = accNameById

	t.usersCache.Update(user.data.ChatId, user)

	accountsMsg := tgbotapi.NewMessage(user.data.ChatId, entity.AccountsMsg)
	accountsMsg.ReplyMarkup = createAccountsMarkup(accounts)

	_, err = t.bot.Send(accountsMsg)
	if err != nil {
		return errors.WithMessagef(err, "send telegram message '%s'", accountsMsg.Text)
	}

	return nil
}

func (t Telegram) selectAccount(ctx context.Context, user User, accountId string) error {
	accName, found := user.data.Invest.AccountNameById[accountId]
	if !found {
		t.logger.Warn(ctx, "selected account not found", log.Any("accountId", accountId))
		return nil
	}

	user.data.Invest.AccountId = accountId
	user.data.Invest.TracingAccountName = accName

	err := t.upUserService(ctx, user)
	if err != nil {
		return errors.WithMessage(err, "up user service")
	}

	return nil
}

func (t Telegram) upUserService(ctx context.Context, user User) error {
	investRepo, err := repository.NewInvest(ctx, user.data.Invest.Config)
	if err != nil {
		return errors.WithMessage(err, "create invest repository")
	}

	tgRepo := repository.NewTgRepository(t.bot, user.data.ChatId)
	userService := service.NewService(ctx, t.logger, investRepo, tgRepo, t.config.OperationsTimeout, user.data.Invest.AccountId)

	user.service = userService
	t.usersCache.Update(user.data.ChatId, user)

	userService.StartTracing()

	keyboardMsg := tgbotapi.NewMessage(user.data.ChatId, entity.StartTracingMsg+" "+user.data.Invest.TracingAccountName)
	keyboardMsg.ReplyMarkup = createStopButtonMarkup()
	_, err = t.bot.Send(keyboardMsg)
	if err != nil {
		return errors.WithMessage(err, "send keyboard message")
	}

	return nil
}

// Cообщение с кнопками: [Название счета]Номер счета(при нажатии)
func createAccountsMarkup(accounts []*investapi.Account) tgbotapi.InlineKeyboardMarkup {
	accRows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, acc := range accounts {
		accRows = append(accRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(acc.GetName(), acc.GetId()),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(accRows...)
}

// Сообщение с кнопкой [Приостановить]
func createStopButtonMarkup() any {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(entity.StopButton, entity.StopButtonCommand),
		),
	)
}

// Сообщение с кнопками [Продолжить] \n [Выбрать другой счет]
func createContinueButtonMarkup() any {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(entity.ContinueButton, entity.ContinueButtonCommand),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(entity.AccountsButton, entity.AccountsButtonCommand),
		),
	)
}
