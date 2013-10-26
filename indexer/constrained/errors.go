package constrained

type PersistenceError struct {
	msg string
}

func NewPersistenceError(msg string) (e *PersistenceError) {
	e = new(PersistenceError)
	e.msg = msg
	return e
}

func (e *PersistenceError) Error() string {
	return e.msg
}
