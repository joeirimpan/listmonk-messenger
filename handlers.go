package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/joeirimpan/listmonk-messenger/messenger"
	"github.com/knadh/listmonk/models"
)

type postback struct {
	Subject     string      `json:"subject"`
	ContentType string      `json:"content_type"`
	Body        string      `json:"body"`
	Recipients  []recipient `json:"recipients"`
	Campaign    *campaign   `json:"campaign"`
}

type campaign struct {
	UUID string   `db:"uuid" json:"uuid"`
	Name string   `db:"name" json:"name"`
	Tags []string `db:"tags" json:"tags"`
}

type recipient struct {
	UUID    string                   `db:"uuid" json:"uuid"`
	Email   string                   `db:"email" json:"email"`
	Name    string                   `db:"name" json:"name"`
	Attribs models.SubscriberAttribs `db:"attribs" json:"attribs"`
	Status  string                   `db:"status" json:"status"`
}

type httpResp struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// handlePostback picks the messager based on url params and pushes message using it.
func handlePostback(w http.ResponseWriter, r *http.Request) {
	var (
		app      = r.Context().Value("app").(*App)
		provider = chi.URLParam(r, "provider")
	)

	// Decode body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "invalid body", http.StatusBadRequest, nil)
		return
	}
	defer r.Body.Close()

	data := &postback{}
	if err := json.Unmarshal(body, &data); err != nil {
		sendErrorResponse(w, "invalid body", http.StatusBadRequest, nil)
		return
	}

	// Get the provider.
	p, ok := app.messengers[provider]
	if !ok {
		sendErrorResponse(w, "unknown provider", http.StatusBadRequest, nil)
		return
	}

	if len(data.Recipients) > 1 {
		sendErrorResponse(w, "invalid recipients", http.StatusBadRequest, nil)
		return
	}

	rec := data.Recipients[0]
	message := messenger.Message{
		Subject:     data.Subject,
		ContentType: data.ContentType,
		Body:        []byte(data.Body),
		Subscriber: models.Subscriber{
			UUID:    rec.UUID,
			Email:   rec.Email,
			Name:    rec.Name,
			Status:  rec.Status,
			Attribs: rec.Attribs,
		},
	}

	if data.Campaign != nil {
		message.Campaign = &models.Campaign{
			UUID: data.Campaign.UUID,
			Name: data.Campaign.Name,
			Tags: data.Campaign.Tags,
		}
	}

	// Send message.
	if err := p.Push(message); err != nil {
		sendErrorResponse(w, "error sending message", http.StatusInternalServerError, nil)
		return
	}

	sendResponse(w, "OK")
}

// wrap is a middleware that wraps HTTP handlers and injects the "app" context.
func wrap(app *App, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "app", app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sendResponse sends a JSON envelope to the HTTP response.
func sendResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	out, err := json.Marshal(httpResp{Status: "success", Data: data})
	if err != nil {
		sendErrorResponse(w, "Internal Server Error", http.StatusInternalServerError, nil)
		return
	}

	w.Write(out)
}

// sendErrorResponse sends a JSON error envelope to the HTTP response.
func sendErrorResponse(w http.ResponseWriter, message string, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	resp := httpResp{Status: "error",
		Message: message,
		Data:    data}
	out, _ := json.Marshal(resp)
	w.Write(out)
}
