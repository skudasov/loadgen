package loadgen

import (
	"encoding/csv"
	"io"
	"os"
	"sync"
)

type CSVData struct {
	Mu        *sync.Mutex
	f         *os.File
	CsvWriter *csv.Writer
	CsvReader *csv.Reader
	Recycle   bool
}

func NewCSVData(f *os.File, recycle bool) *CSVData {
	return &CSVData{
		Mu:        &sync.Mutex{},
		f:         f,
		CsvWriter: csv.NewWriter(f),
		CsvReader: csv.NewReader(f),
		Recycle:   recycle,
	}
}

// RecycleData reads file from the beginning
func (m *CSVData) RecycleData() error {
	_, err := m.f.Seek(0, 0)
	if err != nil {
		return err
	}
	return nil
}

func (m *CSVData) Lock() {
	m.Mu.Lock()
}

func (m *CSVData) Unlock() {
	m.Mu.Unlock()
}

// Read reads string from csv, recycle if EOF
func (m *CSVData) Read() ([]string, error) {
	st, err := m.CsvReader.Read()
	if err == io.EOF {
		if !m.Recycle {
			log.Info("data EOF, not recycling mode, exiting now")
			os.Exit(0)
		}
		if err := m.RecycleData(); err != nil {
			return nil, err
		}
		if st, err = m.CsvReader.Read(); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// Write writes csv string
func (m *CSVData) Write(rec []string) error {
	if err := m.CsvWriter.Write(rec); err != nil {
		return err
	}
	return nil
}

func (m *CSVData) Flush() {
	m.CsvWriter.Flush()
}

func DefaultReadCSV(a Attack) []string {
	lm := a.GetManager()
	s := lm.CsvForHandle(a.GetRunner().Config.ReadFromCsvName)
	s.Lock()
	st, err := s.Read()
	if err != nil {
		s.Unlock()
		log.Fatal(err)
	}
	s.Unlock()
	return st
}

func DefaultWriteCSV(a Attack, data []string) {
	lm := a.GetManager()
	s := lm.CsvForHandle(a.GetRunner().Config.WriteToCsvName)
	s.Lock()
	defer s.Unlock()
	if err := s.Write(data); err != nil {
		log.Fatal(err)
	}
	s.Flush()
}
