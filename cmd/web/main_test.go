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

// Test quiz sequencing.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"inchworks.com/quiz/internal/models"
	"inchworks.com/quiz/internal/quiz"
)

// Control data in globals on page.
type pageGbls struct {
	puppet  string
	page    string
	param   string
	index   string
	update  string
	tick    string
	monitor string
	token   string

	nPage int // page number
}

// Common data to test as quiz controller.

type testingController struct {

	// Safe for goroutines.
	*testing.T
	client       *http.Client
	url          string
	rate         int
	nRounds      int
	nTieBreakers int
	score        chan int // request round to be scored

	// needs serialisation
	muPage      sync.Mutex
	gbls        pageGbls
	scorerToken string
}

// Command line parameters for test
var params struct {
	url          string
	nRounds      int
	nTieBreakers int
	nTeams       int
	rate         int
}

func getParams() {

	// command line flags
	flag.StringVar(&params.url, "url", "http://quiz.local", "Server URL")
	flag.IntVar(&params.nRounds, "rounds", 10, "Number of normal rounds")
	flag.IntVar(&params.nTieBreakers, "ties", 2, "Number of tie breaker rounds")
	flag.IntVar(&params.nTeams, "teams", 8, "Number of teams")
	flag.IntVar(&params.rate, "rate", 2, "Delay between updates (seconds)")
}

func newController(t *testing.T, c *http.Client, url string, rate int, nRounds int, nTieBreakers int) *testingController {

	tc := testingController{
		T:            t,
		client:       c,
		url:          url,
		rate:         rate,
		nRounds:      nRounds,
		nTieBreakers: nTieBreakers,
		score:        make(chan int, 2), // scorers may be two rounds behind
	}

	// client must support cookies
	// No suffix specified, because we are just testing.
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		t.Fatal(err)
	}
	c.Jar = jar

	// start scorers goroutine
	go scorers(&tc)

	return &tc
}

// TestController fetches all slides, as controller, using a private server.
// Scores are generated. This is a quick test of the main application code.
func TestController(t *testing.T) {

	// logging - errors only
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	infoLog := log.New(ioutil.Discard, "", 0)
	threatLog := log.New(ioutil.Discard, "", 0)

	// site configuration, with just environment variables
	cfg := &quiz.Configuration{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		errorLog.Fatal(err)
	}

	// same database as live
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

	// client monitor
	defer app.Monitor.Init()()

	// test server
	ts := httptest.NewServer(app.Routes())
	defer ts.Close()

	// client
	c := ts.Client()

	// all quiz rounds, no delays
	tc := newController(t, c, ts.URL, 0, app.NRounds(), app.NTieBreakers())
	t.Logf("Testing %d rounds, %d tie-breakers", tc.nRounds, tc.nTieBreakers)
	tc.testController(cfg.Option("online"))
	tc.testRestart()
	tc.testRounds()
}

// TestPuppets steps through all slides on a live server.
// It enables a multi-display load test, assuming puppet displays have been opened in real browser windows.
func TestPuppets(t *testing.T) {

	getParams()

	// site configuration, with just environment variables
	cfg := &quiz.Configuration{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		t.Fatal(err)
	}

	// client
	c := &http.Client{}

	// test all quiz rounds, 2 second delay
	tc := newController(t, c, params.url, params.rate, params.nRounds, params.nTieBreakers)
	tc.testController(cfg.Option("online"))
	tc.testRestart()
	tc.testRounds()
}

// testController logs in, and reads the controller page.
func (t *testingController) testController(online bool) {

	var rs *http.Response
	if online {

		// get form to log in
		rs := t.get(t.url + "/user/login")
		t.readTokenController(rs)

		// log in
		data := url.Values{}
		data.Add("username", "admin@example.com")
		data.Add("password", "admin-test-only")
		rs = t.post("/user/login", t.gbls.token, data, "")
		rs.Body.Close()
	}

	// get controllers page, needed for CSRF token (the buttons on the page are forms)
	rs = t.get(t.url + "/controller")
	t.readTokenController(rs)
}

// testNext requests the specified slide number.
func (t *testingController) testNext(nSlide int) {

	// request next page
	data := url.Values{}
	data.Add("next", "1")
	rs := t.post("/control-step", t.gbls.token, data, "")

	// the response is JSON with the page route
	rep := t.readReply(rs)

	// read next page
	rs = t.get(t.url + rep.HRef)
	t.readGlobals(rs)

	// start polling for updates
	// ## ideally should stop and restart this when waiting for scores
	c := make(chan quiz.RepDisplay, 1)
	defer t.controlUpdates(c)()

	// test slide
	t.testSlide(nSlide)

	// wait for scores
	if t.pageType() == models.PageScoresWait {
		resp := <-c

		// request updated page
		t.client.CheckRedirect = nil
		rs = t.get(t.url + resp.HRef)
		t.readGlobals(rs)

		t.testSlide(nSlide)
	}
}

// testRestart starts the quiz and gets the first page.
func (t *testingController) testRestart() {

	// start quiz and get first slide
	data := url.Values{}
	data.Add("display", "C")
	rs := t.post("/control-start", t.gbls.token, data, "/quiz-start")
	t.readGlobals(rs)

	// Start polling for updates. We won't get any in this case.
	c := make(chan quiz.RepDisplay, 1)
	defer t.controlUpdates(c)()

	// pause to allow puppets to update
	if t.rate > 0 {
		time.Sleep(time.Duration(t.rate) * time.Second)
	}

	// starting slide
	t.testSubSlides(0, t.pageType())
}

// Test all quiz rounds

func (t *testingController) testRounds() {

	// start slide
	nSlides := 1

	// questions, answers and scores for normal rounds
	nSlides += t.nRounds * 3

	// final slide
	nSlides += 1

	// QA and scores for tie breakers
	nSlides += t.nTieBreakers * 2

	// final slide again
	nSlides += 1

	// test slides
	for n := 0; n < nSlides; n++ {
		t.testNext(n)
	}

	// ## one more to check we're really done
}

// Test slide

func (t *testingController) testSlide(nSlide int) {

	// pause to allow puppets to update
	if t.rate > 0 {
		time.Sleep(time.Duration(t.rate) * time.Second)
	}

	// step through sub-slides
	t.testSubSlides(nSlide, t.pageType())

	// start scoring as soon as questions shown
	if t.pageType() == models.PageQuestions {

		nRound, err := t.nRound()
		if err != nil {
			t.Errorf("slide %d : %v", nSlide, err)

		} else {

			// initiate scoring, but not for final (sudden death) round
			if nRound < t.nRounds+t.nTieBreakers {
				// initiate scoring
				t.score <- nRound
			}
		}
	}
}

// Test sub-slide

func (t *testingController) testSubSlide(nSlide int, index int) {

	// form with index and touchNav in url.Values
	data := url.Values{}
	data.Add("index", strconv.Itoa(index))
	data.Add("touchNav", "0")

	// notify sub-slide change
	rs := t.post("/control-change", t.gbls.token, data, "")
	rs.Body.Close()

	// ## Could check JSON response, but it never varies

	// brief pause to allow puppets to update
	if t.rate > 0 {
		time.Sleep(time.Duration(t.rate) * time.Second)
	}
}

// Test all sub-slides for slide

func (t *testingController) testSubSlides(nSlide int, nPage int) {

	// approximate number of sub-slides from page type
	nSlides := t.nSubSlides(nPage)

	for i := 1; i <= nSlides; i++ {
		t.testSubSlide(nSlide, i)
	}
}

// ==== Goroutine for polling server ====

// Controller updates. Sends update polling requests to server,

func (t *testingController) controlUpdates(c chan quiz.RepDisplay) func() {

	// monitoring periods
	ticker := time.NewTicker(500 * time.Millisecond)
	quit := make(chan struct{})
	go func() {

		for {
			select {
			case <-ticker.C:
				t.controlUpdate(c)

			case <-quit:
				return
			}
		}
	}()

	// cleanup at end
	return func() {

		// stop the ticker and terminate worker
		close(quit)
		ticker.Stop()
	}
}

// Send controller update request.

func (t *testingController) controlUpdate(c chan quiz.RepDisplay) {

	// update request (reads globals)
	t.muPage.Lock()
	data := url.Values{}
	data.Add("page", t.gbls.page)
	data.Add("param", t.gbls.param)
	data.Add("index", t.gbls.index)
	data.Add("update", t.gbls.update)
	data.Add("second", strconv.Itoa(time.Now().Second()))
	data.Add("monitor", t.gbls.monitor)
	t.muPage.Unlock()

	rs := t.post("/control-update", t.gbls.token, data, "")

	// JSON response
	rep := t.readReply(rs)

	// Check for new page. Index and Tick ignored.
	if rep.HRef != "" {
		c <- *rep
		// ## does this wake main goroutine immediately, or should we terminate controlUpdates?
	}
}

// ==== Support Functions ====

// get sends an HTTP GET request
func (t *testingController) get(url string) *http.Response {

	// get page
	rs, err := t.client.Get(url)
	if err != nil {
		t.Fatal("Error getting page. ", err)
	}

	return rs
}

// Round number (serialised)

func (t *testingController) nRound() (int, error) {

	t.muPage.Lock()
	var param = t.gbls.param
	t.muPage.Unlock()

	n, err := strconv.Atoi(param)
	if err != nil {
		err = fmt.Errorf(`bad param "%s" : %v`, param, err)
	}
	return n, err
}

// Number of sub-slides on page. Doesn't matter if slighly too many.

func (t *testingController) nSubSlides(nPage int) int {

	var n int

	switch nPage {
	case models.PageStatic:
		n = 2
	case models.PageQuestions:
		n = 7
	case models.PageAnswers:
		n = 9 // +2 for interval
	case models.PageScoresWait:
		n = 2
	case models.PageScores:
		n = params.nTeams + 3 // round and leaderboard
	case models.PageFinal:
		n = params.nTeams + 1
	default:
		n = 0
		t.Errorf("unknown page code: %d", nPage)
	}

	return n
}

// Page type (serialised)

func (t *testingController) pageType() int {

	t.muPage.Lock()
	var nPage = t.gbls.nPage
	t.muPage.Unlock()

	return nPage
}

// post sends a POST request with a the CSRF token. It returns the response.
// It is testing two different operations:
// (1) a regular form submission, which should receive a redirected URL,
// (2) an AJAX post, which should receive a JSON reply. The reply may include a new URL.
func (t *testingController) post(path string, token string, data url.Values, toPath string) *http.Response {

	redirect := len(toPath) > 0
	var redirected bool
	if redirect {

		// we're expecting a redirect
		t.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			redirected = true

			// ## check expected redirect?
			/*
				toUrl, err := rs.Location()
				if err != nil {
					t.Errorf("%s Location not returned", path)
				}
			*/

			return nil
		}
	}

	// send request form with token
	data.Add("csrf_token", token)
	rs, err := t.client.PostForm(t.url+path, data)
	if err != nil {
		t.Fatal(err)
	}

	if rs.StatusCode != http.StatusOK {
		t.Errorf("%s status %d; expected %d", path, rs.StatusCode, http.StatusOK)
	}

	// check redirection
	if redirect && !redirected {
		t.Errorf("%s not redirected", path)
	} else if !redirect && redirected {
		t.Errorf("%s was redirected", path)
	}

	return rs
}

// readGlobals scrapes JavaScript values from the page, and closes the response body.
func (t *testingController) readGlobals(rs *http.Response) {

	// read response data in to memory
	body, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		t.Fatal("Error reading HTTP body. ", err)
	}

	// updates globals
	t.muPage.Lock()
	defer t.muPage.Unlock()

	// discard current values
	t.gbls = pageGbls{}

	// regular expression to find globals in page script
	// Tokens include \+, /, = characters, as well as word (\w) characters.
	re := regexp.MustCompile(`gbl(\w+)\s*=\s*"?([\w=/\+]+)"?`) // ## could generalise to something like `(?m)(?P<key>\w+):\s+(?P<value>\w+)$`
	gblDefs := re.FindAllSubmatch(body, -1)

	for _, gblDef := range gblDefs {
		// 3 items in slice : match and 2 submatches
		name := string(gblDef[1])
		v := string(gblDef[2])

		switch name {
		case "Index":
			t.gbls.index = v

		case "Monitor":
			t.gbls.monitor = v

		case "Page":
			t.gbls.page = v

			t.gbls.nPage, err = strconv.Atoi(v)
			if err != nil {
				t.Errorf(`bad page code: "%s"`, v)
				t.gbls.nPage = -1
			}

		case "Param":
			t.gbls.param = v

		case "Puppet":
			t.gbls.puppet = v

		case "Tick":
			t.gbls.tick = v // ## does this match the regexp?

		case "Token":
			t.gbls.token = unescape(v)

		case "Update":
			t.gbls.update = v
		}

		// log globals
		//		t.Log(t.gbls)
		//		fmt.Println(t.gbls)
	}

	rs.Body.Close()
}

// readHRef gets a page path from a JSON response
func (t *testingController) readReply(rs *http.Response) *quiz.RepDisplay {

	// decode JSON response
	var rep quiz.RepDisplay
	if err := json.NewDecoder(rs.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	rs.Body.Close()

	return &rep
}

// readToken scrapes the CSRP token field from the response form, and closes the response body.
func (t *testingController) readToken(rs *http.Response) string {

	// read response data in to memory
	body, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		t.Fatal("Error reading HTTP body. ", err)
	}

	// looking for <input type='hidden' name='csrf_token' value='{{.CSRFToken}}'>
	var token string
	re := regexp.MustCompile(`name='csrf_token' value='(.*?)'`)
	match := re.FindSubmatch(body)
	if match == nil {
		t.Error("Missing CSRF token in form")

	} else if len(match) < 2 {
		t.Errorf("Missing CSRF token value in: %s", string(body))

	} else {
		// 2 items in slice : match and submatch
		token = unescape(string(match[1]))
	}
	rs.Body.Close()

	return token
}

func (t *testingController) readTokenController(rs *http.Response) {

	token := t.readToken(rs)

	// updates globals
	t.muPage.Lock()
	defer t.muPage.Unlock()

	// discard current values
	t.gbls = pageGbls{}

	t.gbls.token = token
}

func (t *testingController) readTokenScorer(rs *http.Response) {

	t.scorerToken = t.readToken(rs)
}

// Scorers goroutine

func scorers(t *testingController) {

	// score rounds until channel closed
	for nRound := range t.score {

		// score all teams, except on tie-break rounds
		round := strconv.Itoa(nRound)
		var nScored int
		if nRound <= t.nRounds {
			nScored = params.nTeams
		} else {
			nScored = 2
		}

		// time limit on scoring (previous answers, scores, answers)
		// (Should be insufficient for R1 and R10)
		nLimit := t.nSubSlides(models.PageQuestions) + // next round questions
			t.nSubSlides(models.PageAnswers) + // previous round answers
			t.nSubSlides(models.PageScores) // previous round scores

		// just in time
		if t.rate > 0 {
			nDelay := nLimit*t.rate - 5
			fmt.Printf("R%d scoring will take %v seconds\n", nRound, nDelay)
			time.Sleep(time.Duration(nDelay) * time.Second)
		}

		// get form for scores
		rs := t.get(t.url + "/score-round/" + round)
		t.readTokenScorer(rs)

		// scores
		data := url.Values{}
		data.Add("nRound", round)
		for i := 0; i < nScored; i++ {

			// random score
			data.Add("index", strconv.Itoa(i))
			data.Add("score", strconv.Itoa(rand.Intn(13)))
		}

		// send form
		rs = t.post("/score-round", t.scorerToken, data, "/scorers")
		t.readTokenScorer(rs)

		if rs.StatusCode != http.StatusOK {
			t.Errorf("score-round %d status %d; expected %d", nRound, rs.StatusCode, http.StatusOK)
		}

		// publish round
		data = url.Values{}
		data.Add("from", "S")
		data.Add("nRound", round)
		rs = t.post("/publish-round", t.scorerToken, data, "/scorers")
		t.readTokenScorer(rs)

		// ## check message on page to see if publication accepted

		fmt.Println("scored round ", round)
	}
}

// unescape replaces escaped '+' characters in a token.
// Somewhere they are escaped, and we have t remove them -(.
func unescape(s string) string {
	return strings.ReplaceAll(s, `&#43;`, `+`)
}
