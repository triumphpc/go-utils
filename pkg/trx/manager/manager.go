package gotrxmanager

import (
	"context"
	"database/sql"
	"fmt"
)

// trxManagerKey - тип для ключа контекста, используемого для хранения транзакции
type trxManagerKey string

// trxKey - конкретный ключ для доступа к транзакции в контексте
const trxKey trxManagerKey = "trxKey"

// NewTransactionManager - конструктор для создания нового менеджера транзакций
func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

// Do - выполняет функцию f в контексте транзакции
// Автоматически обрабатывает начало/коммит/откат транзакции
func (trm *TransactionManager) Do(ctx context.Context, f func(ctx context.Context) (any, error)) (any, error) {
	// Начинаем новую транзакцию
	trx, err := trm.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Добавляем транзакцию в контекст
	ctx = context.WithValue(ctx, trxKey, trx)

	// Выполняем пользовательскую функцию в контексте транзакции
	res, err := f(ctx)
	if err != nil {
		// При ошибке пытаемся откатить транзакцию
		if rbErr := trx.Rollback(); rbErr != nil {
			// Если откат не удался, объединяем ошибки
			err = fmt.Errorf("cannot rollback transaction with err: %s prev error: %s", rbErr, err)
		}
		return nil, err
	}

	// Если все успешно, коммитим транзакцию
	if err := trx.Commit(); err != nil {
		return nil, fmt.Errorf("cannot commit transaction with error: %s", err)
	}

	return res, nil
}

// TxFromContext - извлекает транзакцию из контекста
// Возвращает ошибку если транзакция не найдена или имеет неверный тип
func TxFromContext(ctx context.Context) (*sql.Tx, error) {
	// Получаем значение из контекста по ключу
	t := ctx.Value(trxKey)
	if t == nil {
		return nil, fmt.Errorf("cannot find transaction")
	}

	// Пытаемся привести значение к типу *sql.Tx
	tx, ok := t.(*sql.Tx)
	if !ok {
		return nil, fmt.Errorf("received value is not a *sql.Tx")
	}

	return tx, nil
}
