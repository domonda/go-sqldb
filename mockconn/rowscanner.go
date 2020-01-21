package mockconn

type rowScanner struct{}

func (s *rowScanner) Scan(dest ...interface{}) error {
	return ErrMockedScan
}

func (s *rowScanner) ScanStruct(dest interface{}) error {
	return ErrMockedScan

}
