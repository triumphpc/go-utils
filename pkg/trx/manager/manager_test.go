package gotrxmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestTransactionManager_Do_Success(t *testing.T) {
	// Создаем мок DB
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Ожидаем начало транзакции
	mock.ExpectBegin()

	// Ожидаем коммит транзакции
	mock.ExpectCommit()

	// Создаем менеджер транзакций
	trm := NewTransactionManager(db)

	// Функция, которая будет выполняться в транзакции
	testFunc := func(ctx context.Context) (any, error) {
		// Получаем транзакцию из контекста
		tx, err := TxFromContext(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tx)

		return "success", nil
	}

	// Выполняем операцию
	result, err := trm.Do(context.Background(), testFunc)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_Do_RollbackOnError(t *testing.T) {
	// Создаем мок DB
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Ожидаем начало транзакции
	mock.ExpectBegin()

	// Ожидаем откат транзакции
	mock.ExpectRollback()

	// Создаем менеджер транзакций
	trm := NewTransactionManager(db)

	// Функция, которая вернет ошибку
	testFunc := func(ctx context.Context) (any, error) {
		return nil, errors.New("operation failed")
	}

	// Выполняем операцию
	result, err := trm.Do(context.Background(), testFunc)

	// Проверяем результаты
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "operation failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionManager_Do_BeginTxError(t *testing.T) {
	// Создаем мок DB
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Ожидаем начало транзакции и возвращаем ошибку
	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	// Создаем менеджер транзакций
	trm := NewTransactionManager(db)

	// Функция не должна вызываться при ошибке начала транзакции
	testFunc := func(ctx context.Context) (any, error) {
		t.Fatal("function should not be called")
		return nil, nil
	}

	// Выполняем операцию
	result, err := trm.Do(context.Background(), testFunc)

	// Проверяем результаты
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "begin error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTxFromContext_NotFound(t *testing.T) {
	// Пытаемся получить транзакцию из пустого контекста
	tx, err := TxFromContext(context.Background())

	assert.Nil(t, tx)
	assert.Error(t, err)
	assert.EqualError(t, err, "cannot find transaction")
}

func TestTxFromContext_InvalidType(t *testing.T) {
	// Создаем контекст с неправильным типом значения, но с правильным ключом
	ctx := context.WithValue(context.Background(), trxKey, "not a transaction")

	tx, err := TxFromContext(ctx)

	assert.Nil(t, tx)
	assert.Error(t, err)
	assert.EqualError(t, err, "received value is not a *sql.Tx")
}
