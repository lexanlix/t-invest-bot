package repository

import (
	"fmt"
	"strconv"

	"t-api/entity"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	investapi "github.com/russianinvestments/invest-api-go-sdk/proto"
)

const currencyDecimals = 2

type TgRepository struct {
	bot    *tgbotapi.BotAPI
	chatId int64
}

func NewTgRepository(bot *tgbotapi.BotAPI, chatId int64) *TgRepository {
	return &TgRepository{
		bot:    bot,
		chatId: chatId,
	}
}

//nolint:gosmopolitan
func (r *TgRepository) SendOperations(operations []entity.Operation) error {
	msg := "Операции по счету:\n"

	for _, op := range operations {
		opMsg := op.Type

		if op.HasInstrument {
			opMsg += fmt.Sprintf("\n%s", op.Instrument.GetTicker())
		}

		if op.Quantity != 0 {
			opMsg += fmt.Sprintf("\nКоличество: %d", op.Quantity)
		}

		opMsg += fmt.Sprintf("\nСтоимость: %s", formatMoney(op.Payment))

		if op.Quantity != 0 {
			opMsg += fmt.Sprintf("\nЦена за единицу: %s", formatMoney(op.Price))
		}

		opMsg += fmt.Sprintf("\nДата операции: %s", op.Date.AsTime().Local().Format("2006-01-02 15:04:05"))

		state := mapState(op.GetState())
		if len(state) != 0 {
			opMsg += fmt.Sprintf("\nСтатус: %s", state)
		}

		msg += "\n" + opMsg + "\n"
	}

	_, err := r.bot.Send(tgbotapi.NewMessage(r.chatId, msg))
	if err != nil {
		return errors.WithMessage(err, "send message")
	}

	return nil
}

func formatMoney(v *investapi.MoneyValue) string {
	currency := entity.LabelByCode(v.GetCurrency())
	return fmt.Sprintf("%d.%s %s", v.GetUnits(), formatNano(v.GetNano(), v.GetNano() < 0), currency)
}

func formatNano(nano int32, isNeg bool) string {
	if isNeg {
		nano = -nano
	}

	strNano := strconv.Itoa(int(nano))

	if len(strNano) < currencyDecimals {
		return strNano
	}

	return strNano[:2]
}

func mapState(state investapi.OperationState) string {
	switch state {
	case investapi.OperationState_OPERATION_STATE_UNSPECIFIED:
		return "Не определен"
	case investapi.OperationState_OPERATION_STATE_EXECUTED:
		return "Исполнена"
	case investapi.OperationState_OPERATION_STATE_CANCELED:
		return "Отменена"
	case investapi.OperationState_OPERATION_STATE_PROGRESS:
		return "Исполняется"
	}

	return ""
}
