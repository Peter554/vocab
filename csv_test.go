package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Csv_Export(t *testing.T) {
	db := memoryDb(t)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 3,
		PracticeAt:     inDays(2),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)

	csv := NewCsv(db)

	var buf bytes.Buffer
	err := csv.Export(&buf)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(&buf)
	require.Nil(t, err)

	expected := fmt.Sprintf(`term,translation,knowledge_level,practice_at
foo1,bar1,3,%s
foo2,bar2,1,%s
`, inDaysJSON(2), inDaysJSON(1))
	require.Equal(t, expected, string(data))
}

func Test_Csv_Import(t *testing.T) {
	cases := []string{
		fmt.Sprintf(`term,translation,knowledge_level,practice_at
foo1,bar1,3,%s
foo2,bar2,1,%s
`, inDaysJSON(2), inDaysJSON(1)),
		fmt.Sprintf(`practice_at,translation,term,knowledge_level
%s,bar1,foo1,3
%s,bar2,foo2,1
`, inDaysJSON(2), inDaysJSON(1)),
	}

	for _, data := range cases {
		db := memoryDb(t)

		dbResult := db.Create(&Vocab{
			Term:           "hello",
			Translation:    "world",
			KnowledgeLevel: 1,
			PracticeAt:     inDays(1),
		})
		require.Nil(t, dbResult.Error)

		csv := NewCsv(db)

		err := csv.Import(strings.NewReader(data))
		require.Nil(t, err)

		var count int64
		dbResult = db.Model(&Vocab{}).Count(&count)
		require.Nil(t, dbResult.Error)
		require.Equal(t, int64(3), count)

		vocabs := make([]Vocab, 0)
		dbResult = db.Find(&vocabs)
		require.Nil(t, dbResult.Error)

		require.Equal(t, uint(1), vocabs[0].ID)
		require.Equal(t, "hello", vocabs[0].Term)
		require.Equal(t, "world", vocabs[0].Translation)
		require.Equal(t, uint(1), vocabs[0].KnowledgeLevel)
		require.True(t, vocabs[0].PracticeAt.Equal(inDays(1)))

		require.Equal(t, uint(2), vocabs[1].ID)
		require.Equal(t, "foo1", vocabs[1].Term)
		require.Equal(t, "bar1", vocabs[1].Translation)
		require.Equal(t, uint(3), vocabs[1].KnowledgeLevel)
		require.True(t, vocabs[1].PracticeAt.Equal(inDays(2)))

		require.Equal(t, uint(3), vocabs[2].ID)
		require.Equal(t, "foo2", vocabs[2].Term)
		require.Equal(t, "bar2", vocabs[2].Translation)
		require.Equal(t, uint(1), vocabs[2].KnowledgeLevel)
		require.True(t, vocabs[2].PracticeAt.Equal(inDays(1)))
	}

}

func Test_Csv_Import_Clean(t *testing.T) {
	db := memoryDb(t)

	dbResult := db.Create(&Vocab{
		Term:           "hello",
		Translation:    "world",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)

	data := strings.NewReader(fmt.Sprintf(`term,translation,knowledge_level,practice_at
foo1,bar1,3,%s
foo2,bar2,1,%s
`, inDaysJSON(2), inDaysJSON(1)))

	csv := NewCsv(db)

	err := csv.ImportClean(data)
	require.Nil(t, err)

	var count int64
	dbResult = db.Model(&Vocab{}).Count(&count)
	require.Nil(t, dbResult.Error)
	require.Equal(t, int64(2), count)

	vocabs := make([]Vocab, 0)
	dbResult = db.Find(&vocabs)
	require.Nil(t, dbResult.Error)

	require.Equal(t, uint(1), vocabs[0].ID)
	require.Equal(t, "foo1", vocabs[0].Term)
	require.Equal(t, "bar1", vocabs[0].Translation)
	require.Equal(t, uint(3), vocabs[0].KnowledgeLevel)
	require.True(t, vocabs[0].PracticeAt.Equal(inDays(2)))

	require.Equal(t, uint(2), vocabs[1].ID)
	require.Equal(t, "foo2", vocabs[1].Term)
	require.Equal(t, "bar2", vocabs[1].Translation)
	require.Equal(t, uint(1), vocabs[1].KnowledgeLevel)
	require.True(t, vocabs[1].PracticeAt.Equal(inDays(1)))
}

func Test_ExportImport(t *testing.T) {
	db := memoryDb(t)

	dbResult := db.Create(&Vocab{
		Term:           "foo1",
		Translation:    "bar1",
		KnowledgeLevel: 3,
		PracticeAt:     inDays(2),
	})
	require.Nil(t, dbResult.Error)
	dbResult = db.Create(&Vocab{
		Term:           "foo2",
		Translation:    "bar2",
		KnowledgeLevel: 1,
		PracticeAt:     inDays(1),
	})
	require.Nil(t, dbResult.Error)

	csv := NewCsv(db)

	var buf bytes.Buffer
	err := csv.Export(&buf)
	require.Nil(t, err)

	err = csv.Import(&buf)
	require.Nil(t, err)

	var count int64
	dbResult = db.Model(&Vocab{}).Count(&count)
	require.Nil(t, dbResult.Error)
	require.Equal(t, int64(4), count)
}

func Test_Import_ErrMissingHeading(t *testing.T) {
	db := memoryDb(t)

	data := strings.NewReader(fmt.Sprintf(`term,knowledge_level,practice_at
foo1,3,%s
foo2,1,%s
`, inDaysJSON(2), inDaysJSON(1)))

	csv := NewCsv(db)

	err := csv.Import(data)
	require.ErrorIs(t, err, ErrMissingHeading{Heading: "translation"})
}

func Test_Import_ErrBadRow(t *testing.T) {
	db := memoryDb(t)

	data := strings.NewReader(fmt.Sprintf(`term,translation,knowledge_level,practice_at
foo1,bar1,3,%s
foo2,bar2,a,%s
`, inDaysJSON(2), inDaysJSON(1)))

	csv := NewCsv(db)

	err := csv.Import(data)
	require.ErrorIs(t, err, ErrBadRow{Number: 3, Field: "knowledge_level"})

	var count int64
	dbResult := db.Model(&Vocab{}).Count(&count)
	require.Nil(t, dbResult.Error)
	require.Equal(t, int64(0), count)
}
