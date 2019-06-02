package handler

import (
	"net/http"
	"time"
)

func createCookie(w http.ResponseWriter, name, value string) {
	cookie := &http.Cookie{
		Name:    name,
		Value:   value,
		Path:    "/",
		Expires: time.Now().Add(cookieDays * time.Hour * 24),
	}
	http.SetCookie(w, cookie)
}

func refreshCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(cookieDays * time.Hour * 24)
	http.SetCookie(w, cookie)
}

func deleteCookie(w http.ResponseWriter, cookie *http.Cookie) {
	cookie.Path = "/"
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}
