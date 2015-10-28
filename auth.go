package main

import (
	"fmt"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/objx"
	"log"
	"net/http"
	"strings"
)

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("auth"); err == http.ErrNoCookie {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else if err != nil {
		panic(err.Error())
	} else {
		h.next.ServeHTTP(w, r)
	}
}
func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	segs := strings.Split(r.URL.Path, "/")
	action := segs[2]
	provider := segs[3]
	switch action {
	case "callback":
		providerobj, err := gomniauth.Provider(provider)
		log.Println("providerobj: ", providerobj)
		if err != nil {
			log.Fatalln("Error when trying to get provider", provider, "-", err)
		}

		omap, err := objx.FromURLQuery(r.URL.RawQuery)
		log.Println(omap)
		if err != nil {
			log.Fatalln("Error when trying to get object from query", provider, "-", err)
		}

		creds, err := providerobj.CompleteAuth(omap)
		log.Println("creds: ", creds, err)
		if err != nil {
			log.Fatalln("Error when trying to complete auth for", provider, "-", err)
		}

		log.Println("retrieving user")
		user, err := providerobj.GetUser(creds)
		log.Println("user: ", user)
		if err != nil {
			log.Fatalln("Error when trying to get user from", provider, "-", err)
		}

		authCookieValue := objx.New(map[string]interface{}{
			"name":       user.Name(),
			"avatar_url": user.AvatarURL(),
		}).MustBase64()
		http.SetCookie(w, &http.Cookie{
			Name:  "auth",
			Value: authCookieValue,
			Path:  "/"})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	case "login":
		providerobj, err := gomniauth.Provider(provider)
		if err != nil {
			log.Fatalln("Error when trying to get the provider", provider, "-", err)
		}
		loginUrl, err := providerobj.GetBeginAuthURL(nil, nil)
		if err != nil {
			log.Fatalln("error when trying to GetBeginAuthURL for", provider, "-", err)
		}
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "auth",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.Header()["Location"] = []string{"/chat"}
	w.WriteHeader(http.StatusTemporaryRedirect)
}
