//go:build cgo

package security

func init() {
	isEncrypted = true
}

func newCipherSessionStore(dbPath, passphrase string) (SessionStore, error) {
	return NewSQLCipherSessionStore(dbPath, passphrase)
}
