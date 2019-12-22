package persist

import (
	"github.com/mattn/go-sqlite3"
)

type Persistable interface {
	Persist(tx Transaction) error
}

type Transaction interface {
	Insert(list ...interface{}) error
}

type InsertFunc func(...interface{}) error

func (f InsertFunc) Insert(list ...interface{}) error {
	return f(list...)
}

func InsertIgnoringDupes(t Transaction) Transaction {
	return InsertFunc(func(list ...interface{}) error {
		err := t.Insert(list...)
		if sqliteError, ok := err.(sqlite3.Error); ok {
			if sqliteError.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil // silently ignore
			}
		}
		return err
	})
}
