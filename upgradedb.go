//
// Copyright (c) 2019 Ted Unangst <tedu@tedunangst.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"database/sql"
	"os"
)

var myVersion = 43

type dbexecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func doordie(db dbexecer, s string, args ...interface{}) {
	_, err := db.Exec(s, args...)
	if err != nil {
		elog.Fatalf("can't run %s: %s", s, err)
	}
}

func upgradedb() {
	db := opendatabase()
	dbversion := 0
	getconfig("dbversion", &dbversion)
	getconfig("servername", &serverName)

	if dbversion < 40 {
		elog.Fatal("database is too old to upgrade")
	}
	switch dbversion {
	case 40:
		doordie(db, "PRAGMA journal_mode=WAL")
		blobdb := openblobdb()
		doordie(blobdb, "PRAGMA journal_mode=WAL")
		doordie(db, "update config set value = 41 where key = 'dbversion'")
		fallthrough
	case 41:
		tx, err := db.Begin()
		if err != nil {
			elog.Fatal(err)
		}
		rows, err := tx.Query("select honkid, noise from honks where format = 'markdown' and precis <> ''")
		if err != nil {
			elog.Fatal(err)
		}
		m := make(map[int64]string)
		var dummy Honk
		for rows.Next() {
			err = rows.Scan(&dummy.ID, &dummy.Noise)
			if err != nil {
				elog.Fatal(err)
			}
			precipitate(&dummy)
			m[dummy.ID] = dummy.Noise
		}
		rows.Close()
		for id, noise := range m {
			_, err = tx.Exec("update honks set noise = ? where honkid = ?", noise, id)
			if err != nil {
				elog.Fatal(err)
			}
		}
		err = tx.Commit()
		if err != nil {
			elog.Fatal(err)
		}
		doordie(db, "update config set value = 42 where key = 'dbversion'")
		fallthrough
	case 42:
		doordie(db, "update honks set what = 'honk', flags = flags & ~ 32 where what = 'tonk' or what = 'wonk'")
		doordie(db, "delete from honkmeta where genus = 'wonkles' or genus = 'guesses'")
		doordie(db, "update config set value = 43 where key = 'dbversion'")
		fallthrough
	case 43:

	default:
		elog.Fatalf("can't upgrade unknown version %d", dbversion)
	}
	os.Exit(0)
}
