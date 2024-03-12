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

package quiz

import (
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golangcollege/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/justinas/nosurf"
	"github.com/microcosm-cc/bluemonday"

	"github.com/inchworks/usage"
	"github.com/inchworks/webparts/etx"
	"github.com/inchworks/webparts/limithandler"
	"github.com/inchworks/webparts/monitor"
	"github.com/inchworks/webparts/multiforms"
	"github.com/inchworks/webparts/stack"
	"github.com/inchworks/webparts/uploader"
	"github.com/inchworks/webparts/users"

	"inchworks.com/quiz/internal/models"
	"inchworks.com/quiz/internal/models/mysql"
	"inchworks.com/quiz/internal/setup"
	"inchworks.com/quiz/web"
)

// file locations on server
var (
	CertPath  = "../certs" // cached certificates
	MediaPath = "../media" // location of audio, pictures and videos
	SetupPath = "../setup" // exported setup for server
	SitePath  = "../site"  // site-specific resources
)

// Site configuration
type Configuration struct {

	// domains served via HTTPS
	Domains   []string `yaml:"domains" env:"domains" env-default:""`
	CertEmail string   `yaml:"certificate-email" env:"certificate-email" env-default:""`

	// from command line only
	AddrHTTP  string `yaml:"http-addr" env:"http" env-default:":8000" env-description:"HTTP address"`
	AddrHTTPS string `yaml:"https-addr" env:"https" env-default:":4000" env-description:"HTTPS address"`

	Secret string `yaml:"session-secret" env:"session-secret" env-default:"Qa8>zGrrg4cfERxdgVWGxoDG3eqcpijT" env-description:"Secret key for sessions"`

	// new DSN
	DBSource   string `yaml:"db-source" env:"db-source" env-default:"tcp(quiz_db:3306)/quiz"`
	DBUser     string `yaml:"db-user" env:"db-user" env-default:"server"`
	DBPassword string `yaml:"db-password" env:"db-password" env-default:"<server-password>"`

	// administrator
	AdminName     string `yaml:"admin-name" env:"admin-name" env-default:""`
	AdminPassword string `yaml:"admin-password" env:"admin-password" env-default:"<your-password>"`

	// image sizes
	MaxW      int `yaml:"image-width" env-default:"1920"` // maximum stored image dimensions
	MaxH      int `yaml:"image-height" env-default:"1200"`
	ThumbW    int `yaml:"thumbnail-width" env-default:"278"` // thumbnail size
	ThumbH    int `yaml:"thumbnail-height" env-default:"208"`
	MaxUpload int `yaml:"max-upload" env-default:"512"` // maximum file upload (megabytes)

	// operational settings
	MaxUploadAge    time.Duration `yaml:"max-upload-age" env:"max-upload-age" env-default:"1h"` // maximum time for a round update. Units m or h.
	MonitorInterval int           `yaml:"monitor-interval" env-default:"5000"`                  // monitor display (mS)
	SlideItems      int           `yaml:"slide-items" env-default:"10"`                         // default maximum items per slide
	TopTeams        int           `yaml:"top-teams" env-default:"0"`                            // number of top teams on leaderboard

	// variants
	Options       []string      `yaml:"options" env:"options" env-default:""`                             // site features: online, remote, RPi
	AudioTypes    []string      `yaml:"audio-types" env:"audio-types" env-default:".mp3,.aac,.flac,.m4a"` // audio types
	VideoSnapshot time.Duration `yaml:"video-snapshot"  env-default:"3s"`                                 // snapshot time within video. -ve for no snapshots.
	VideoPackage  string        `yaml:"video-package" env:"video-package" env-default:"ffmpeg"`           // video processing package
	VideoTypes    []string      `yaml:"video-types" env:"video-types" env-default:".mp4,.mov"`            // video types

	// from environment only
	TestSelf bool `env:"test-self" env-default:"false"`
}

// Operation to update media for round.
type OpUpdateRound struct {
	RoundId int64
	tx      etx.TxId
}

// Application struct supplies application-wide dependencies.
type Application struct {
	Version string

	cfg       *Configuration
	hasRemote bool
	isOnline  bool

	errorLog  *log.Logger
	infoLog   *log.Logger
	threatLog *log.Logger
	session   *sessions.Session
	templates map[string]*template.Template

	db *sqlx.DB
	tx *sqlx.Tx

	QuestionStore  *mysql.QuestionStore
	QuizStore      *mysql.QuizStore
	redoStore      *mysql.RedoStore
	ResponseStore  *mysql.ResponseStore
	RoundStore     *mysql.RoundStore
	ScoreStore     *mysql.ScoreStore
	ContestStore   *mysql.ContestStore
	TeamStore      *mysql.TeamStore
	statisticStore *mysql.StatisticStore
	userStore      *mysql.UserStore

	// common components
	lhs      *limithandler.Handlers
	recorder *usage.Recorder
	tm       *etx.TM
	uploader *uploader.Uploader
	users    users.Users

	// HTML sanitizer for questions and answers, etc.
	sanitizer *bluemonday.Policy

	// Since we show one quiz at a time, we can cache state here.
	// With a public web server, we'd need a per-quiz cache.
	displayState DisplayState
	quizState    QuizState

	// Channels to background worker
	chRound chan OpUpdateRound
	chDone  chan bool

	// private components
	Monitor  monitor.Monitor
	StaticFS fs.FS

	// slow to evaluate, so worth caching
	NTeams     int
	teamAccess map[string]string
}

// New returns a quiz application, common to live and test
func New(cfg *Configuration, errorLog *log.Logger, infoLog *log.Logger, threatLog *log.Logger, db *sqlx.DB) *Application {

	// redirect to test folders
	testSite := os.Getenv("testSite")
	if testSite != "" {
		CertPath = filepath.Join(testSite, filepath.Base(CertPath))
		MediaPath = filepath.Join(testSite, filepath.Base(MediaPath))
		SitePath = filepath.Join(testSite, filepath.Base(SitePath))
	}

	// package templates
	var pts []fs.FS

	// templates for user management
	pt, err := fs.Sub(users.WebFiles, "web/template")
	if err != nil {
		errorLog.Fatal(err)
	}
	pts = append(pts, pt)

	// application templates
	forApp, err := fs.Sub(web.Files, "template")
	if err != nil {
		errorLog.Fatal(err)
	}

	// initialise template cache
	templates, err := stack.NewTemplates(pts, forApp, os.DirFS(filepath.Join(SitePath, "templates")), templateFuncs)
	if err != nil {
		errorLog.Fatal(err)
	}

	// session manager
	clientSess := sessions.New([]byte(cfg.Secret))
	clientSess.Lifetime = 12 * time.Hour

	// dependency injection
	app := &Application{
		cfg:       cfg,
		errorLog:  errorLog,
		infoLog:   infoLog,
		threatLog: threatLog,
		session:   clientSess,
		templates: templates,
		db:        db,
		sanitizer: bluemonday.UGCPolicy(),
	}

	// main options
	app.hasRemote = app.cfg.Option("remote")
	if app.hasRemote {
		app.isOnline = true // must be an online server if we have remote teams
	} else {
		app.isOnline = app.cfg.Option("online")
	}

	// export setup files for Raspberry Pi server
	if app.cfg.Option("RPi") {
		infoLog.Print("Exporting setup files for RPi")
		err = setup.Export("rpi", SetupPath)
		if err != nil {
			app.errorLog.Print(err)
		}
	}

	// embedded static files from packages
	staticForms, err := fs.Sub(multiforms.WebFiles, "web/static")
	if err != nil {
		errorLog.Fatal(err)
	}

	// embedded static files from app
	staticApp, err := fs.Sub(web.Files, "static")
	if err != nil {
		errorLog.Fatal(err)
	}
	staticUploader, err := fs.Sub(uploader.WebFiles, "web/static")
	if err != nil {
		errorLog.Fatal(err)
	}

	// combine embedded static files with site customisation
	// ## perhaps site resources should be under "static"?
	app.StaticFS, err = stack.NewFS(staticForms, staticUploader, staticApp, os.DirFS(SitePath))
	if err != nil {
		errorLog.Fatal(err)
	}

	// ## not sure if these child objects make the code clearer
	app.displayState.app = app
	app.quizState.app = app

	// intialise data stores
	quiz, quizSess := app.initStores(cfg)

	// set up extended transaction manager, and recover
	app.tm = etx.New(app, app.redoStore)

	// setup media upload processing
	app.uploader = &uploader.Uploader{
		FilePath:     MediaPath,
		MaxW:         app.cfg.MaxW,
		MaxH:         app.cfg.MaxH,
		ThumbW:       app.cfg.ThumbW,
		ThumbH:       app.cfg.ThumbH,
		MaxAge:       app.cfg.MaxUploadAge,
		SnapshotAt:   app.cfg.VideoSnapshot,
		AudioTypes:   app.cfg.AudioTypes,
		VideoPackage: app.cfg.VideoPackage,
		VideoTypes:   app.cfg.VideoTypes,
	}
	app.uploader.Initialise(app.errorLog, &app.quizState, app.tm)

	if app.isOnline {

		// setup statistics recording, with defaults
		if app.recorder, err = usage.New(app.statisticStore, usage.Daily, 0, 0, 0, 0, 0); err != nil {
			errorLog.Fatal(err)
		}

		// initialise rate limiter
		app.lhs = limithandler.Start(8*time.Hour, 24*time.Hour)

		// user management
		app.users = users.Users{
			App:   app,
			Roles: []string{"unknown", "audience", "team", "organiser", "admin"},
			Store: app.userStore,
		}
	}

	// setup cached quiz state
	app.quizState.Init(quiz, app.RoundStore.Count())
	app.displayState.Init(quizSess)
	app.cacheTeams()

	// make worker channels
	app.chDone = make(chan bool, 1) // closing this signals worker goroutines to return
	app.chRound = make(chan OpUpdateRound, 10)

	// start background worker
	go app.quizState.worker(app.chRound, app.chDone)

	// redo any pending operations
	infoLog.Print("Starting operation recovery")
	if err := app.tm.Recover(&app.quizState, app.uploader); err != nil {
		errorLog.Fatal(err)
	}

	if app.cfg.TestSelf {
		infoLog.Print("** Ready for Testing **")
	}

	return app
}

// NewServerLogger returns a logger that filter common events cause by background noise from internet idiots.
// (Typically probes using unsupported TLS versions or attempting HTTPS connection without a domain name.
// Also continuing access attempts with the domain of a previous holder of the server's IP address.)
func (app *Application) NewServerLog(out io.Writer, prefix string, flag int) *log.Logger {

	filter := []string{"TLS handshake error"}

	return app.recorder.NewLogger(out, prefix, flag, filter, "bad-https")
}

// Stop makes a clean shutdown of the application.
func (app *Application) Stop() {

	if app.isOnline {
		app.lhs.Stop()
		app.recorder.Stop()
	}
	close(app.chDone)
}

// ** INTERFACE FUNCTIONS FOR WEBPARTS/USERS **

// Authenticated adds a logged-in user's ID to the contest.
func (app *Application) Authenticated(r *http.Request, id int64) {
	app.session.Put(r, "authenticatedUserID", id)
}

// Flash adds a confirmation message to the next page, via the contest.
func (app *Application) Flash(r *http.Request, msg string) {
	app.session.Put(r, "flash", msg)
}

// GetRedirect returns the next page after log-in, probably from a contest key.
func (app *Application) GetRedirect(r *http.Request) string { return "/" }

// Log optionally records an error.
func (app *Application) Log(err error) {
	app.errorLog.Print(err)
}

// LogThreat optionally records a rejected request to sign-up or log-in.
func (app *Application) LogThreat(msg string, r *http.Request) {
	// not needed for this application
}

// OnAddUser is called to add any additional application data for a user.
func (app *Application) OnAddUser(user *users.User) {
	// not needed for this application
}

// OnRemoveUser is called to delete any application data for a user.
func (app *Application) OnRemoveUser(tx etx.TxId, user *users.User) {
	// not needed for this application
}

// Render writes an HTTP response using the specified template and field (embedded as Users).
func (app *Application) Render(w http.ResponseWriter, r *http.Request, template string, usersField interface{}) {
	app.render(w, r, template, &usersFormData{Users: usersField})
}

// Rollback specifies that the transaction started by Serialise be cancelled.
func (app *Application) Rollback() {
	app.quizState.rollbackTx = true
}

// Serialise optionally requests application-level serialisation.
// If updates=true, the store is to be updated and a transaction might be started (especially if a user is to be added or deleted).
// The returned function will be called at the end of the operation.
func (app *Application) Serialise(updates bool) func() {
	return app.displayState.updatesAll()
}

// Token returns a token to be added to the form as the hidden field csrf_token.
func (app *Application) Token(r *http.Request) string {
	return nosurf.Token(r)
}

// ** FUNCTIONS FOR TESTING **

func (app *Application) NRounds() int {

	return app.quizState.nFullRounds
}

func (app *Application) NTieBreakers() int {

	return app.quizState.nTieRounds
}

// Initialise data stores

func (app *Application) initStores(cfg *Configuration) (*models.Quiz, *models.Contest) {

	defer app.quizState.updatesQuiz()()

	// setup stores, with reference to a common transaction
	app.QuestionStore = mysql.NewQuestionStore(app.db, &app.tx, app.errorLog)
	app.QuizStore = mysql.NewQuizStore(app.db, &app.tx, app.errorLog)
	app.redoStore = mysql.NewRedoStore(app.db, &app.tx, app.errorLog)
	app.ResponseStore = mysql.NewResponseStore(app.db, &app.tx, app.errorLog)
	app.RoundStore = mysql.NewRoundStore(app.db, &app.tx, app.errorLog)
	app.ScoreStore = mysql.NewScoreStore(app.db, &app.tx, app.errorLog)
	app.ContestStore = mysql.NewContestStore(app.db, &app.tx, app.errorLog)
	app.TeamStore = mysql.NewTeamStore(app.db, &app.tx, app.errorLog)
	app.userStore = mysql.NewUserStore(app.db, &app.tx, app.errorLog)

	// uses own transaction
	app.statisticStore = mysql.NewStatisticStore(app.db, app.errorLog)

	// fast display refresh for local server, slower for online
	var refresh int
	if app.isOnline {
		refresh = 3 // 4s
	} else {
		refresh = 1 // 1s
	}

	// setup new database and administrator, if needed, and get quiz record
	quiz, sess, err := mysql.Setup(app.QuizStore, app.ContestStore, app.userStore, 1, cfg.AdminName, cfg.AdminPassword, refresh)
	if err != nil {
		app.errorLog.Fatal(err)
	}

	// save quiz ID for other stores that need it
	app.RoundStore.QuizId = quiz.Id
	app.ScoreStore.QuizId = quiz.Id
	app.TeamStore.QuizId = quiz.Id

	return quiz, sess
}
