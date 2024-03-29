package main

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/klev-dev/klev-api-go"
	"github.com/klev-dev/klev-api-go/logs"
	"github.com/klev-dev/klev-api-go/messages"
	flag "github.com/spf13/pflag"
)

type App struct {
	templateFiles  fs.FS
	templateReload bool
	templates      *template.Template

	logs     *logs.Client
	messages *messages.Client
}

func main() {
	reload := flag.Bool("reload", false, "if uses dynamic reloading")
	flag.Parse()

	if err := run(*reload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(reload bool) error {
	t, err := getTemplates(reload)
	if err != nil {
		return err
	}

	cfg := klev.NewConfig(os.Getenv("KLEV_TOKEN_DEMO"))

	a := &App{
		templateFiles:  t,
		templateReload: reload,

		logs:     logs.New(cfg),
		messages: messages.New(cfg),
	}

	srv := http.Server{
		Addr:    "127.0.0.1:8000",
		Handler: a,
	}
	return srv.ListenAndServe()
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := a.mux(w, r); err != nil {
		fmt.Println("err:", err)
		// TODO
	}
}

func (a *App) mux(w http.ResponseWriter, r *http.Request) error {
	switch r.URL.Path {
	case "/login":
		return a.login(w, r)
	}

	userCookie, err := r.Cookie("user")
	if errors.Is(err, http.ErrNoCookie) {
		return a.redirect(w, r, "/login")
	}
	user := userCookie.Value

	switch r.URL.Path {
	case "/":
		return a.index(w, r, user)
	case "/addroom":
		return a.addRoom(w, r, user)
	default:
		return a.room(w, r, user)
	}
}

func (a *App) login(w http.ResponseWriter, r *http.Request) error {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return err
		}
		if username := r.FormValue("username"); username != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "user",
				Value:    username,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
			})
		}
		return a.redirect(w, r, "/")
	}
	return a.render(w, "login", nil)
}

func (a *App) index(w http.ResponseWriter, r *http.Request, user string) error {
	rooms, err := a.logs.List(r.Context())
	if err != nil {
		return err
	}

	if len(rooms) == 0 {
		room, err := a.logs.Create(r.Context(), klev.LogCreateParams{
			Metadata:    "General",
			TrimSeconds: 60 * 60,
		})
		if err != nil {
			return err
		}
		return a.redirect(w, r, fmt.Sprintf("/%s", room.LogID))
	}

	for _, room := range rooms {
		if room.Metadata == "General" {
			return a.redirect(w, r, fmt.Sprintf("/%s", room.LogID))
		}
	}

	return a.redirect(w, r, fmt.Sprintf("/%s", rooms[0].LogID))
}

func (a *App) addRoom(w http.ResponseWriter, r *http.Request, user string) error {
	if r.Method != http.MethodPost {
		return a.redirect(w, r, "/")
	}

	if err := r.ParseForm(); err != nil {
		return err
	}
	if name := r.FormValue("room-name"); name != "" {
		room, err := a.logs.Create(r.Context(), klev.LogCreateParams{
			Metadata:    name,
			TrimSeconds: 60 * 60,
		})
		if err != nil {
			return err
		}
		return a.redirect(w, r, fmt.Sprintf("/%s", room.LogID))
	}

	return a.redirect(w, r, "/")
}

type RoomMessage struct {
	From string
	When time.Time
	Text string
}

func (a *App) room(w http.ResponseWriter, r *http.Request, user string) error {
	logID, err := klev.ParseLogID(r.URL.Path[1:])
	if err != nil {
		return err
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return err
		}
		if text := r.FormValue("message"); text != "" {
			_, err := a.messages.Publish(r.Context(), logID,
				[]klev.PublishMessage{klev.NewPublishMessage(user, text)})
			switch {
			case klev.IsErrLogNotFound(err):
				return a.redirect(w, r, "/")
			case err != nil:
				return err
			}
		}

		return a.redirect(w, r, fmt.Sprintf("/%s", logID))
	}

	var msgs []RoomMessage
	offset := klev.OffsetOldest
	for {
		next, messages, err := a.messages.Consume(r.Context(), logID,
			klev.ConsumeOffset(offset), klev.ConsumeLen(32))
		switch {
		case klev.IsErrLogNotFound(err):
			return a.redirect(w, r, "/")
		case err != nil:
			return err
		}

		if next == offset || len(messages) == 0 {
			break
		}
		offset = next

		for _, msg := range messages {
			msgs = append(msgs, RoomMessage{
				From: string(msg.Key),
				When: msg.Time,
				Text: string(msg.Value),
			})
		}
	}

	logs, err := a.logs.List(r.Context())
	if err != nil {
		return err
	}

	return a.render(w, "room", map[string]interface{}{
		"user":  user,
		"logID": logID,
		"logs":  logs,
		"msgs":  msgs,
	})
}

func (a *App) redirect(w http.ResponseWriter, r *http.Request, target string) error {
	http.Redirect(w, r, target, http.StatusFound)
	return nil
}

func (a *App) render(w http.ResponseWriter, name string, data any) error {
	if a.templates == nil || a.templateReload {
		t := template.New("")
		t, err := t.ParseFS(a.templateFiles, "*.html")
		if err != nil {
			return err
		}
		a.templates = t
	}
	return a.templates.ExecuteTemplate(w, fmt.Sprintf("%s.html", name), data)
}
