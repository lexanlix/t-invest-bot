package entity

//nolint:gochecknoglobals,gosec
var (
	StartCommand = "/start"

	StopButton        = "Приостановить отслеживание"
	StopButtonCommand = "/stop"

	ContinueButton        = "Продолжить отслеживание"
	ContinueButtonCommand = "/continue"

	AccountsButton        = "Выбрать другой счет"
	AccountsButtonCommand = "/set_account"

	GiveTokenMsg    = "Для регистрации отправьте токен"
	AccountsMsg     = "Получены инвестиционные счета. Выберите счет, по которому будут отслеживаться операции"
	StartTracingMsg = "Начато отслеживание операций по счету"
)
