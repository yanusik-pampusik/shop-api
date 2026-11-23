package service

import "errors"

var (
	// ErrNotFound — запись не найдена (превратится в 404).
	ErrNotFound = errors.New("запись не найдена")

	// ErrInvalidInput — некорректные входные данные (превратится в 400).
	ErrInvalidInput = errors.New("некорректные входные данные")

	// ErrConflict — конфликт (например, дубликат). Превратится в 409.
	ErrConflict = errors.New("конфликт данных")
)