package entity

func LabelByCode(code string) string {
	labelByCode := map[string]string{
		"usd": "$",
		"eur": "€",
		"rub": "₽",
	}

	label, ok := labelByCode[code]
	if ok {
		return label
	}

	return code
}
