package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type Csv struct {
	db *gorm.DB
}

func NewCsv(db *gorm.DB) *Csv {
	return &Csv{
		db: db,
	}
}

func (c *Csv) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)

	err := csvWriter.Write([]string{"term", "translation", "knowledge_level", "practice_at"})
	if err != nil {
		return err
	}

	vocabs := make([]Vocab, 0)
	dbResult := c.db.Find(&vocabs)
	if dbResult.Error != nil {
		return dbResult.Error
	}

	for _, vocab := range vocabs {
		err := csvWriter.Write([]string{
			vocab.Term,
			vocab.Translation,
			strconv.Itoa(int(vocab.KnowledgeLevel)),
			vocab.PracticeAt.Format(time.RFC3339)})
		if err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return nil
}

func (c *Csv) Import(r io.Reader) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		return c.doImport(tx, r)
	})
}

func (c *Csv) ImportClean(r io.Reader) error {
	return c.db.Transaction(func(tx *gorm.DB) error {
		dbResult := tx.Where("1 = 1").Delete(&Vocab{})
		if dbResult.Error != nil {
			return dbResult.Error
		}
		return c.doImport(tx, r)
	})
}

func (c *Csv) doImport(tx *gorm.DB, r io.Reader) error {
	csvReader := csv.NewReader(r)

	headings, err := csvReader.Read()
	if err != nil {
		return err
	}

	for _, heading := range []string{
		"term",
		"translation",
		"knowledge_level",
		"practice_at",
	} {
		if indexOf(headings, heading) < 0 {
			return ErrMissingHeading{Heading: heading}
		}
	}

	rows, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	for idx, row := range rows {
		dRow, err := dictRow(row, headings)
		if err != nil {
			return ErrBadRow{Number: idx + 2}
		}

		knowledgeLevel, err := strconv.Atoi(dRow["knowledge_level"])
		if err != nil {
			return ErrBadRow{Number: idx + 2, Field: "knowledge_level"}
		}

		praticeAt, err := time.Parse(time.RFC3339, dRow["practice_at"])
		if err != nil {
			return ErrBadRow{Number: idx + 2, Field: "practice_at"}
		}

		vocab := &Vocab{
			Term:           dRow["term"],
			Translation:    dRow["translation"],
			KnowledgeLevel: uint(knowledgeLevel),
			PracticeAt:     praticeAt,
		}
		dbResult := tx.Create(vocab)
		if dbResult.Error != nil {
			return dbResult.Error
		}
	}
	return nil
}

type ErrMissingHeading struct {
	Heading string
}

func (e ErrMissingHeading) Error() string {
	return "Missing heading. heading: " + e.Heading
}

type ErrBadRow struct {
	Number int
	Field  string
}

func (e ErrBadRow) Error() string {
	return fmt.Sprintf("Bad row. number: %d, field: %s", e.Number, e.Field)
}

func indexOf(a []string, k string) int {
	for idx, s := range a {
		if s == k {
			return idx
		}
	}
	return -1
}

func dictRow(row []string, headings []string) (map[string]string, error) {
	if len(row) != len(headings) {
		return nil, errors.New("Row length mismatch")
	}
	m := make(map[string]string)
	for idx, heading := range headings {
		m[heading] = row[idx]
	}
	return m, nil
}
