package txnmgr

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
)

type DBConn interface {
	Exec(string, ...interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Tx interface {
	driver.Tx
	End() error
}

type TxnMgr struct {
	db       *sql.DB
	tx       *sql.Tx
	done     bool
	endhooks []func()
}

type rootTx struct {
	mgr *TxnMgr
}

func (rt *rootTx) Commit() error {
	if rt.mgr.done {
		return fmt.Errorf("transaction already end")
	}

	return rt.mgr.commit()
}

func (rt *rootTx) Rollback() error {
	if rt.mgr.done {
		return nil
	}

	return rt.mgr.rollback()
}

func (rt *rootTx) End() error {
	if rt.mgr.done {
		return nil
	}
	return rt.mgr.rollback()
}

type nestTx struct {
	mgr            *TxnMgr
	isCommitCalled bool
}

func (nt *nestTx) Commit() error {
	if nt.mgr.done {
		return fmt.Errorf("transaction already end")
	}

	nt.isCommitCalled = true

	return nil
}

func (nt *nestTx) Rollback() error {
	if nt.mgr.done {
		return nil
	}

	return nt.mgr.rollback()
}

func (nt *nestTx) End() error {
	if !nt.isCommitCalled {
		return nt.Rollback()
	}
	return nil
}

func (tm *TxnMgr) Begin() (Tx, error) {
	if tm.tx != nil {
		return &nestTx{
			mgr:            tm,
			isCommitCalled: false,
		}, nil
	}

	tx, err := tm.db.Begin()
	if err != nil {
		return nil, err
	}

	tm.done = false
	tm.tx = tx

	return &rootTx{tm}, nil
}

func (tm *TxnMgr) commit() error {
	tm.done = true
	tm.tx = nil

	if err := tm.tx.Commit(); err != nil {
		return err
	}

	for _, fn := range tm.endhooks {
		fn()
	}
	tm.endhooks = make([]func(), 0, 3)

	return nil
}

func (tm *TxnMgr) rollback() error {
	tm.done = true
	tm.tx = nil
	tm.endhooks = make([]func(), 0, 3)

	return tm.tx.Rollback()
}

func (tm *TxnMgr) DBC() DBConn {
	if tm.tx != nil {
		return tm.tx
	}
	return tm.db
}

func (tm *TxnMgr) AddEndhook(fn func()) {
	tm.endhooks = append(tm.endhooks, fn)
}

func NewTxnMgr(db *sql.DB) *TxnMgr {
	return &TxnMgr{
		db:       db,
		endhooks: make([]func(), 0, 3),
	}
}
