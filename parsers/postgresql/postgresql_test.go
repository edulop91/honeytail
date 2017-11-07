package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/honeycombio/honeytail/event"
	"github.com/stretchr/testify/assert"
)

func TestSingleQueryParsing(t *testing.T) {
	testcases := []struct {
		in       string
		expected event.Event
	}{
		{
			in: `2017-11-07 00:05:16 UTC [3053-3] postgres@postgres LOG:  duration: 0.681 ms  statement: SELECT d.datname as "Name",
	       pg_catalog.pg_get_userbyid(d.datdba) as "Owner",
	       pg_catalog.pg_encoding_to_char(d.encoding) as "Encoding",
	       d.datcollate as "Collate",
	       d.datctype as "Ctype",
	       pg_catalog.array_to_string(d.datacl, E'\n') AS "Access privileges"
	FROM pg_catalog.pg_database d
	ORDER BY 1;`,
			expected: event.Event{
				Timestamp: time.Date(2017, 11, 7, 0, 5, 16, 0, time.UTC),
				Data: map[string]interface{}{
					"user":             "postgres",
					"database":         "postgres",
					"duration":         0.681,
					"pid":              3053,
					"session_id":       3,
					"query":            "SELECT d.datname as \"Name\", pg_catalog.pg_get_userbyid(d.datdba) as \"Owner\", pg_catalog.pg_encoding_to_char(d.encoding) as \"Encoding\", d.datcollate as \"Collate\", d.datctype as \"Ctype\", pg_catalog.array_to_string(d.datacl, E'\\n') AS \"Access privileges\" FROM pg_catalog.pg_database d ORDER BY 1;",
					"normalized_query": "select d.datname as ?, pg_catalog.pg_get_userbyid(d.datdba) as ?, pg_catalog.pg_encoding_to_char(d.encoding) as ?, d.datcollate as ?, d.datctype as ?, pg_catalog.array_to_string(d.datacl, e?) as ? from pg_catalog.pg_database d order by ?;",
				},
			},
		},
	}

	for _, tc := range testcases {
		in := make(chan []string)
		out := make(chan event.Event)
		p := Parser{}
		p.Init(nil)
		go p.handleEvents(in, out)
		in <- strings.Split(tc.in, "\n")
		close(in)
		got := <-out
		assert.Equal(t, got, tc.expected)
	}
}

func TestMultipleQueryParsing(t *testing.T) {
	in := `
2017-11-07 01:43:18 UTC [3542-5] postgres@test LOG:  duration: 9.263 ms  statement: INSERT INTO test (id, name, value) VALUES (1, 'Alice', 'foo');
2017-11-07 01:43:27 UTC [3542-6] postgres@test LOG:  duration: 0.841 ms  statement: INSERT INTO test (id, name, value) VALUES (2, 'Bob', 'bar');
2017-11-07 01:43:39 UTC [3542-7] postgres@test LOG:  duration: 15.577 ms  statement: SELECT * FROM test
	WHERE id=1;
2017-11-07 01:43:42 UTC [3542-8] postgres@test LOG:  duration: 0.501 ms  statement: SELECT * FROM test
	WHERE id=2;
`
	out := []event.Event{
		event.Event{
			Timestamp: time.Date(2017, 11, 7, 1, 43, 18, 0, time.UTC),
			Data: map[string]interface{}{
				"user":             "postgres",
				"database":         "test",
				"duration":         9.263,
				"pid":              3542,
				"session_id":       5,
				"query":            "INSERT INTO test (id, name, value) VALUES (1, 'Alice', 'foo');",
				"normalized_query": "insert into test (id, name, value) values (?, ?, ?);",
			},
		},
		event.Event{
			Timestamp: time.Date(2017, 11, 7, 1, 43, 27, 0, time.UTC),
			Data: map[string]interface{}{
				"user":             "postgres",
				"database":         "test",
				"duration":         0.841,
				"pid":              3542,
				"session_id":       6,
				"query":            "INSERT INTO test (id, name, value) VALUES (2, 'Bob', 'bar');",
				"normalized_query": "insert into test (id, name, value) values (?, ?, ?);",
			},
		},
		event.Event{
			Timestamp: time.Date(2017, 11, 7, 1, 43, 39, 0, time.UTC),
			Data: map[string]interface{}{
				"user":             "postgres",
				"database":         "test",
				"duration":         15.577,
				"pid":              3542,
				"session_id":       7,
				"query":            "SELECT * FROM test WHERE id=1;",
				"normalized_query": "select * from test where id=?;",
			},
		},
		event.Event{
			Timestamp: time.Date(2017, 11, 7, 1, 43, 42, 0, time.UTC),
			Data: map[string]interface{}{
				"user":             "postgres",
				"database":         "test",
				"duration":         0.501,
				"pid":              3542,
				"session_id":       8,
				"query":            "SELECT * FROM test WHERE id=2;",
				"normalized_query": "select * from test where id=?;",
			},
		},
	}

	parser := Parser{}
	inChan := make(chan string)
	sendChan := make(chan event.Event, 4)
	go parser.ProcessLines(inChan, sendChan, nil)
	for _, line := range strings.Split(in, "\n") {
		inChan <- line
	}
	close(inChan)
	for _, expected := range out {
		got := <-sendChan
		assert.Equal(t, got, expected)
	}
}