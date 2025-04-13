package gotrxmanager

import "database/sql"

// TransactionManager - менеджер транзакций, оборачивающий соединение с БД
type TransactionManager struct {
	db *sql.DB
}
