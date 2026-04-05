package repository

type DBError struct {
	Err              error
	Value            string
	HTTPResponseCode int
}

func (de *DBError) Error() string {
	return de.Err.Error()
}
