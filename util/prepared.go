package util

import (
	"database/sql"
	"fmt"
	"strings"
)

// Prepared is tool for preparing sql statements and the using them.
type Prepared struct {
	commands map[string]*sql.Stmt
	finished bool
}

// NewPrepared initializes inner map.
func NewPrepared() *Prepared {
	return &Prepared{commands: make(map[string]*sql.Stmt)}
}

func (p *Prepared) Prepare(db *sql.DB, id, content string) error {
	return p.LoadFileWithCUstomSyntax(db, id, content, "--+", "+--")
}

// LoadFile loads a sql command file with given start end syntax. The example file:
//
//	--+command1+--
//	CREATE TABLE something
// 	--+command2+--
//
// 	DROP TABLE something
// This file can be parsed with start = '--+' and end = '+--'. For each string where str != ""
// will method prepend the id followed by ˙:˙ followed by command name.
func (p *Prepared) LoadFileWithCUstomSyntax(db *sql.DB, id, content, start, end string) error {
	if p.finished {
		panic("cannot add file to finished Prepared")
	}
	if id != "" {
		id += ":"
	}
	parts := strings.Split(content, start)
	for i, part := range parts {
		if part == "" {
			continue
		}
		name, query := SliceOn(part, end)
		if name == "" || query == "" {
			return fmt.Errorf("statement no %d under id %s is malformed", i, id)
		}
		if name == "init" {
			_, err := db.Exec(query)
			if err != nil {
				return fmt.Errorf("error executing init statement %d under id %s: %s", i, id, err)
			}
		}
		stmt, err := db.Prepare(query)
		if err != nil {
			return fmt.Errorf("statement %s under %s cannot be prepared: %s", name, id, err)
		}
		name = id + name
		if _, ok := p.commands[name]; ok {
			return fmt.Errorf("statement %s is already registered", name)
		}
		p.commands[name] = stmt
	}

	return nil
}

// Finish makes p finished.
func (p *Prepared) Finish() {
	p.finished = true
}

// Finished returns true if you are no longer able to add Prepared statements.
func (p *Prepared) Finished() bool {
	return p.finished
}

// Get returns command under the id or nil if it does not exist.
func (p *Prepared) Get(id string) *sql.Stmt {
	return p.commands[id]
}

func SliceOn(base, div string) (a, b string) {
	limit := len(base) - len(div)
	for i := 0; i < limit; i++ {
		if base[i:i+len(div)] == div {
			return base[:i], base[i+len(div):]
		}
	}

	return base, ""
}
