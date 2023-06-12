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
	"bytes"
	"database/sql"
	notrand "math/rand"
	"sync"
	"time"

	"humungus.tedunangst.com/r/webs/gate"
)

type Doover struct {
	ID   int64
	When time.Time
	Rcpt string
	Msgs [][]byte
}

func sayitagain(goarounds int64, userid int64, doover Doover) {
	rcpt := doover.Rcpt
	var drift time.Duration
	switch goarounds {
	case 1:
		drift = 5 * time.Minute
	case 2:
		drift = 1 * time.Hour
	case 3:
		drift = 4 * time.Hour
	case 4:
		drift = 12 * time.Hour
	case 5:
		drift = 24 * time.Hour
	default:
		ilog.Printf("he's dead jim: %s", rcpt)
		return
	}
	drift += time.Duration(notrand.Int63n(int64(drift / 10)))
	when := time.Now().Add(drift)
	data := bytes.Join(doover.Msgs, []byte{0})
	_, err := stmtAddDoover.Exec(when.UTC().Format(dbtimeformat), goarounds, userid, rcpt, data)
	if err != nil {
		elog.Printf("error saving doover: %s", err)
	}
	select {
	case pokechan <- 0:
	default:
	}
}

var dqmtx sync.Mutex

func delinquent(rcpt string, msg []byte) bool {
	dqmtx.Lock()
	defer dqmtx.Unlock()
	row := stmtDeliquentCheck.QueryRow(rcpt)
	var dooverid int64
	var data []byte
	err := row.Scan(&dooverid, data)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		elog.Printf("error scanning deliquent check: %s", err)
		return true
	}
	data = append(data, 0)
	data = append(data, msg...)
	_, err = stmtDeliquentUpdate.Exec(data, dooverid)
	if err != nil {
		elog.Printf("error updating deliquent: %s", err)
		return true
	}
	return true
}

func deliverate(goarounds int64, userid int64, rcpt string, msg []byte) {
	if delinquent(rcpt, msg) {
		return
	}
	var d Doover
	d.Rcpt = rcpt
	d.Msgs = append(d.Msgs, msg)
	deliveration(goarounds, userid, d)
}

var garage = gate.NewLimiter(40)

func deliveration(goarounds int64, userid int64, doover Doover) {
	garage.Start()
	defer garage.Finish()

	var ki *KeyInfo
	ok := ziggies.Get(userid, &ki)
	if !ok {
		elog.Printf("lost key for delivery")
		return
	}
	var inbox string
	rcpt := doover.Rcpt
	// already did the box indirection
	if rcpt[0] == '%' {
		inbox = rcpt[1:]
	} else {
		var box *Box
		ok := boxofboxes.Get(rcpt, &box)
		if !ok {
			ilog.Printf("failed getting inbox for %s", rcpt)
			sayitagain(goarounds+1, userid, doover)
			return
		}
		inbox = box.In
	}
	for i, msg := range doover.Msgs {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		err := PostMsg(ki.keyname, ki.seckey, inbox, msg)
		if err != nil {
			ilog.Printf("failed to post json to %s: %s", inbox, err)
			doover.Msgs = doover.Msgs[i:]
			sayitagain(goarounds+1, userid, doover)
			return
		}
	}
}

var pokechan = make(chan int, 1)

func getdoovers() []Doover {
	rows, err := stmtGetDoovers.Query()
	if err != nil {
		elog.Printf("wat?")
		time.Sleep(1 * time.Minute)
		return nil
	}
	defer rows.Close()
	var doovers []Doover
	for rows.Next() {
		var d Doover
		var dt string
		err := rows.Scan(&d.ID, &dt)
		if err != nil {
			elog.Printf("error scanning dooverid: %s", err)
			continue
		}
		d.When, _ = time.Parse(dbtimeformat, dt)
		doovers = append(doovers, d)
	}
	return doovers
}

func redeliverator() {
	sleeper := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-pokechan:
			if !sleeper.Stop() {
				<-sleeper.C
			}
			time.Sleep(5 * time.Second)
		case <-sleeper.C:
		}

		doovers := getdoovers()

		now := time.Now()
		nexttime := now.Add(24 * time.Hour)
		for _, d := range doovers {
			if d.When.Before(now) {
				var goarounds, userid int64
				var data []byte
				dqmtx.Lock()
				row := stmtLoadDoover.QueryRow(d.ID)
				err := row.Scan(&goarounds, &userid, &d.Rcpt, &data)
				if err != nil {
					elog.Printf("error scanning doover: %s", err)
					dqmtx.Unlock() // defer
					continue
				}
				_, err = stmtZapDoover.Exec(d.ID)
				if err != nil {
					elog.Printf("error deleting doover: %s", err)
					dqmtx.Unlock() // defer
					continue
				}
				dqmtx.Unlock() // defer
				d.Msgs = bytes.Split(data, []byte{0})
				rcpt := d.Rcpt
				ilog.Printf("redeliverating %s try %d", rcpt, goarounds)
				deliveration(goarounds, userid, d)
			} else if d.When.Before(nexttime) {
				nexttime = d.When
			}
		}
		now = time.Now()
		dur := 5 * time.Second
		if now.Before(nexttime) {
			dur += nexttime.Sub(now).Round(time.Second)
		}
		sleeper.Reset(dur)
	}
}
