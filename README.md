# txnmgr

transaction manager

# usage

```go
import (
  "github.com/mizkei/txnmgr"
)

func Do(mgr *txnmgr.TxnMgr) error {
  txn, err := mgr.Begin()
  if err != nil {
    return err
  }
  defer txn.End()

  _, err = mgr.DBC().Exec(`insert into user (name) values(?)`, "test")
  if err != nil {
    return err
  }

  if err := txn.Commit(); err != nil {
    return err
  }

  return nil
}
```
