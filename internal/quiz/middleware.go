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

// ## This is mostly common with PicInch, but I don't know how to reorganise it

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/inchworks/usage"
	"github.com/inchworks/webparts/v2/users"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"

	"inchworks.com/quiz/internal/models"
)

// put context key in its own type,
// to avoid collision with any 3rd-party packages using request context
type contextKey string

const contextKeyUser = contextKey("authenticatedUser")

type AuthenticatedUser struct {
	id   int64
	role int
}

// HTTP handlers

// authenticate returns a handler to check if this is an authenticated user or not.
// It checks any ID against the database, to see if this is still a valid user since the last login.
func (app *Application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// check for authenticated user in contest
		exists := app.session.Exists(r, "authenticatedUserID")
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		// check user against database
		user, err := app.userStore.Get(app.session.Get(r, "authenticatedUserID").(int64))
		if errors.Is(err, models.ErrNoRecord) || user.Status < users.UserActive {
			app.session.Remove(r, "authenticatedUserID")
			next.ServeHTTP(w, r)
			return
		} else if err != nil {
			app.log(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// copy the request with indicator that user is authenticated
		auth := AuthenticatedUser{
			id:   user.Id,
			role: user.Role,
		}
		ctx := context.WithValue(r.Context(), contextKeyUser, auth)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// limitFile returns a handler to limit file requests, per user.
func (app *Application) limitFile(next http.Handler) http.Handler {

	// no limit - but can be set to block all file requests after other bad requests
	lh := app.lhs.New("F", 0, 0, 20, "", next)

	lh.SetReportHandler(func(r *http.Request, addr string, status string) {

		app.threatLog.Printf("%s - %s file requests, too many after %s", addr, status, r.RequestURI)
	})

	return lh
}

// limitLogin restricts login (and signup) rates.
//
// 50s per attempt, with an initial burst of 20, banned after 10 rejects.
func (app *Application) limitLogin(next http.Handler) http.Handler {

	// (For comparison, Fail2Ban defaults are to jail for 10 minutes, ban after just 3 attempts within 10 minutes).

	lh := app.lhs.New("L", time.Minute, 5, 15, "", next)

	lh.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		http.Error(w, "Too many failed attempts - wait a few minutes", http.StatusTooManyRequests)
	}))

	lh.SetReportHandler(func(r *http.Request, addr string, status string) {

		// try to get the username
		username := "unknown"
		if r.ParseForm() == nil {
			username = r.PostForm.Get("username")
		}

		app.threatLog.Printf("%s - %s login, too many for user \"%s\"", addr, status, username)
	})

	return lh
}

// limitPage returns a handler to limit web page requests, per user.
func (app *Application) limitPage(next http.Handler) http.Handler {

	// 2 per second with burst of 5, banned after 20 rejects,
	// (This is too restrictive to be applied to file requests.)
	lim := app.lhs.New("P", 500*time.Millisecond, 5, 20, "", next)

	lim.SetReportHandler(func(r *http.Request, addr string, status string) {

		app.threatLog.Printf("%s - %s page requests, too many after %s", addr, status, r.RequestURI)
	})

	return lim
}

// logRequest counts a page request in statistics.
func (app *Application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// anonymise request
		// ## could be more selective and include some IDs
		request := strings.SplitN(r.URL.RequestURI(), "/", 3)
		if len(request) < 2 {
			request[1] = "nil"
		}

		// usage statistics
		rec := app.recorder
		rec.Count(request[1], "page")
		userId := app.authenticatedUser(r)
		if userId != 0 {
			rec.Seen(rec.FormatID("U", userId), "user")
		} else {
			if ip := usage.FormatIP(r.RemoteAddr); ip != "" {
				app.recorder.Seen(ip, "visitor")
			}
		}

		next.ServeHTTP(w, r)
	})
}

// noQuery blocks probes with random query parameters,
// (mainly so we don't count them as valid visitors).
func (app *Application) noQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			app.threat("bad query", r)
			http.Error(w, "Query parameters not accepted", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// noSurf adds CSRF protection.
func (app *Application) noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{

		// cookie can't be secure if we are in offline mode or testing
		HttpOnly: app.isOnline && !app.cfg.TestSelf,
		Path:     "/",
		Secure:   app.isOnline && !app.cfg.TestSelf,
	})

	return csrfHandler
}

// offlineHeaders sets HTTP headers for offline (local access) web pages and resources
func (app *Application) offlineHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// publicHeaders sets HTTP headers for public web pages and resources
func (app *Application) publicHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// set canical URL for search engines, if we accept more than one domain
		if len(app.cfg.Domains) > 1 {
			u := *r.URL
			u.Host = app.cfg.Domains[1] // first listed domain
			u.Scheme = "https"
			w.Header().Set("Link", `<`+u.String()+`>; rel="canonical"`)
		}

		w.Header().Set("Cache-Control", "public, max-age=600")
		next.ServeHTTP(w, r)
	})
}

// recoverPanic allows the server to continue after a panic. It is set in httprouter.
func (app *Application) recoverPanic() func(http.ResponseWriter, *http.Request, interface{}) {

	return func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.Header().Set("Connection", "close")
		app.log(fmt.Errorf("%s", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// requireAdmin specifies that administrator authentication is needed for access to this page.
func (app *Application) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.authAs(w, r, models.UserAdmin) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuthentication specifies that minimum authentication is needed, typically to log out
func (app *Application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.authAs(w, r, models.UserAudience) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireOrganiser specifies that authentication as a quiz organiser is needed for access to this page.
func (app *Application) requireOrganiser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.authAs(w, r, models.UserOrganiser) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requirePuppet specifies that a visitor access token or organiser authentication is needed for access to this page.
func (app *Application) requirePuppet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// access token supplied as page parameter
		t := httprouter.ParamsFromContext(r.Context()).ByName("puppet")
		if len(t) > 1 {
			if !app.checkAccessPuppet(t) {
				app.threat("bad visitor code", r)
				http.Redirect(w, r, "/no-access", http.StatusSeeOther)
				return
			}

			// pages that require authentication should not be cached by browser
			w.Header().Set("Cache-Control", "no-store")

			// no token - must be a quiz organiser
		} else if !app.authAs(w, r, models.UserOrganiser) {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireTeam specifies that a team access token or organiser authentication is needed for access to this page.
func (app *Application) requireTeam(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// access token supplied as page parameter
		t := httprouter.ParamsFromContext(r.Context()).ByName("access")
		if len(t) > 1 {
			if !app.checkAccessTeam(t, httprouter.ParamsFromContext(r.Context()).ByName("nTeam")) {
				app.threat("bad team code", r)
				http.Redirect(w, r, "/no-access", http.StatusSeeOther)
				return
			}

			// pages that require authentication should not be cached by browser
			w.Header().Set("Cache-Control", "no-store")

			// no token - must be a quiz organiser
		} else if !app.authAs(w, r, models.UserOrganiser) {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// routeNotFound returns a handler that logs and rate limits HTTP requests to non-existent routes.
// Typically these are intrusion attempts. Not called for non-existent files :-).
func (app *Application) routeNotFound() http.Handler {

	// allow 1 every 10 minutes, burst of 3, banned after 1 rejection,
	// (typically probing for vulnerable PHP files).
	lim := app.lhs.New("R", 10*time.Minute, 3, 1, "F,P", nil)

	lim.SetReportHandler(func(r *http.Request, addr string, status string) {

		app.threatLog.Printf("%s - %s for bad requests, after %s", addr, status, r.RequestURI)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ignore some common bad requests, so we don't ban unreasonably
		d, f := path.Split(r.URL.Path)
		if d == "/" && path.Ext(f) == ".png" {
			app.threat("no favicon", r)
			http.NotFound(w, r) // possibly a favicon for an ancient mobile device
			return
		}

		ok, status := lim.Allow(r)
		if ok {
			app.threat("bad URL", r)
			http.NotFound(w, r)
		} else {
			http.Error(w, "Intrusion attempt suspected", status)
		}
	})
}

// secureHeaders adds HTTP headers for security against XSS and Clickjacking.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Frame-Options", "deny")

		next.ServeHTTP(w, r)
	})
}

// wwwRedirect redirects a request for "www.domain" to "domain"
func wwwRedirect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if host := strings.TrimPrefix(r.Host, "www."); host != r.Host {
			// Request host has www. prefix. Redirect to host with www. trimmed.
			u := *r.URL
			u.Host = host
			u.Scheme = "https"
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// File handler

// noDirFileSystem implements a file system that blocks browse access to a folder.
// It allows index.html to be served as default.
type noDirFileSystem struct {
	fs http.FileSystem
}

func (nfs noDirFileSystem) Open(path string) (http.File, error) {

	// From https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings

	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}

// Helper function

// requireAuthAs implements a handler to check for authentication in the specified role
func (app *Application) authAs(w http.ResponseWriter, r *http.Request, minRole int) bool {

	if !app.isAuthenticated(r, minRole) {
		if app.isAuthenticated(r, models.UserUnknown) {
			http.Error(w, "User is not authorised for role", http.StatusUnauthorized)
		} else {
			app.session.Put(r, "redirectPathAfterLogin", r.URL.Path)
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		}
		return false
	}

	// pages that require authentication should not be cached by browser
	w.Header().Set("Cache-Control", "no-store")
	return true
}

// threat records an attempted intrusion.
func (app *Application) threat(event string, r *http.Request) {
	app.threatLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())

	rec := app.recorder
	if rec != nil {
		rec.Count(event, "threat")
		rec.Seen(usage.FormatIP(r.RemoteAddr), "suspect")
	}
}
