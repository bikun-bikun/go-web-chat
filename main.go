package main

import (
	"flag"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/objx"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"

	"gopkg.in/yaml.v2"
)

var avatars Avatar = UseFileSystemAvatar

type config struct {
	SecurityKey string      `yaml:"securityKey"`
	Google      oauthConfig `yaml:"google"`
	GitHub      oauthConfig `yaml:"github"`
	Facebook    oauthConfig `yaml:"facebook"`
}

type oauthConfig struct {
	ClientId    string `yaml:"clientId"`
	Secret      string `yaml:"secret"`
	RedirectUri string `yaml:"redirectUri"`
}

var (
	envFilePath = "env.yml"
	conf        config
)

type templateHandler struct {
	once     sync.Once
	filename string
	tmpl     *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.tmpl =
			template.Must(template.ParseFiles(filepath.Join("templates",
				t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.tmpl.Execute(w, data)
}

func main() {

	buf, err := ioutil.ReadFile(envFilePath)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(buf, &conf)
	if err != nil {
		panic(err)
	}

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse()

	gomniauth.SetSecurityKey(conf.SecurityKey)
	gomniauth.WithProviders(
		facebook.New(conf.Facebook.ClientId, conf.Facebook.Secret, conf.Facebook.RedirectUri),
		github.New(conf.GitHub.ClientId, conf.GitHub.Secret, conf.GitHub.RedirectUri),
		google.New(conf.Google.ClientId, conf.Google.Secret, conf.Google.RedirectUri),
	)

	r := newRoom()
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "auth",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/upload", &templateHandler{filename: "upload.html"})
	http.HandleFunc("/uploader", uploaderHandler)
	http.Handle("/avatars/", http.StripPrefix("/avatars/", http.FileServer(http.Dir("./avatars"))))

	go r.run()

	//webサーバの開始
	log.Println("webサーバを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServer:", err)
	}
}
