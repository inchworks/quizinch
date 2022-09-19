// Copyright © Rob Burke inchworks.com, 2020.

// This file is part of QuizInch.
//
// QuizInch is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// QuizInch is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with QuizInch.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jmoiron/sqlx"

	"github.com/inchworks/webparts/server"
	"inchworks.com/quiz/internal/quiz"
)

// version and copyright
const (
	version = "0.4.11"
	notice  = `
	Copyright (C) Rob Burke inchworks.com, 2020.
	This website software comes with ABSOLUTELY NO WARRANTY.
	This is free software, and you are welcome to redistribute it under certain conditions.
	For details see the license on https://github.com/inchworks/quizinch.
`
)

func main() {

	// "main() should parse flags, open connections to databases, loggers, and such, then hand off execution to a high level object."

	// logging
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	threatLog := log.New(os.Stdout, "THREAT\t", log.Ldate|log.Ltime)
	infoLog.Printf("Inchworks Quiz %s", version)
	infoLog.Print(notice)

	// redirect to test folders
	test := os.Getenv("test-path")
	if test != "" {
		quiz.CertPath = filepath.Join(test, filepath.Base(quiz.CertPath))
		quiz.MediaPath = filepath.Join(test, filepath.Base(quiz.MediaPath))
		quiz.SetupPath = filepath.Join(test, filepath.Base(quiz.SetupPath))
		quiz.SitePath = filepath.Join(test, filepath.Base(quiz.SitePath))
	}

	// site configuration
	cfg := &quiz.Configuration{}
	if err := cleanenv.ReadConfig(filepath.Join(quiz.SitePath, "configuration.yml"), cfg); err != nil {

		// no file - go with just environment variables
		infoLog.Print(err.Error())
		if err := cleanenv.ReadEnv(cfg); err != nil {
			errorLog.Fatal(err)
		}
	}

	// database
	dsn := fmt.Sprintf("%s:%s@%s?parseTime=true", cfg.DBUser, cfg.DBPassword, cfg.DBSource)
	db, err := openDB(dsn)
	if err != nil {
		errorLog.Fatal(err)
	} else {
		infoLog.Print("Connected to database")
	}

	// close DB on exit
	defer db.Close()

	// initialise application
	app := quiz.New(cfg, errorLog, infoLog, threatLog, db)
	defer app.Stop()
	app.Version = version

	// client monitor
	defer app.Monitor.Init()()

	// preconfigured HTTP/HTTPS server
	srv := &server.Server{

		ErrorLog: app.NewServerLog(os.Stdout, "SERVER\t", log.Ldate|log.Ltime),
		InfoLog:  infoLog,

		CertEmail: cfg.CertEmail,
		CertPath:  quiz.CertPath,
		Domains:   cfg.Domains,

		// port addresses
		AddrHTTP:  cfg.AddrHTTP,
		AddrHTTPS: cfg.AddrHTTPS,
	}

	srv.Serve(app)
}

// Open database
func openDB(dsn string) (db *sqlx.DB, err error) {

	// ## jmoiron/sqlx recommends github.com/mattn/go-sqlite3

	// Running under Docker, the DB container may not be ready yet - retry for 30s
	nRetries := 30

	for ; nRetries > 0; nRetries-- {
		db, err = sqlx.Open("mysql", dsn)
		if err == nil {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}

	// test a connection to DB
	for ; nRetries > 0; nRetries-- {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}

	if nRetries == 0 {
		return nil, err
	} else {
		return db, nil
	}
}
