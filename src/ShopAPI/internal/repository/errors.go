package repository

import "errors"

var ErrConflict = errors.New("конфликт данных")
var ErrInsufficientStock = errors.New("недостаточно товара")
