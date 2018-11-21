package main

import (
	"flag"
	"fmt"
	"github.com/stretchr/gomniauth"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	Google   oauthConfig `yaml:"google"`
	GitHub   oauthConfig `yaml:"github"`
	Facebook oauthConfig `yaml:"facebook"`
}

type oauthConfig struct {
	ClientId    string `yaml:"clientId"`
	Secret      string `yaml:"secret"`
	RedirectUri string `yaml:"redirectUri"`
}

var (
	configrationPath = "env.yml"
	conf             config
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
	t.tmpl.Execute(w, r)
}

func main() {

	buf, err := ioutil.ReadFile(configrationPath)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(buf, &conf)
	if err != nil {
		panic(err)
	}

	fmt.Println(conf.Google.ClientId)

	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse()

	gomniauth.SetSecurityKey("hogehoge1234")
	gomniauth.WithProviders(
		facebook.New("", "", ""),
		github.New("", "", ""),
		google.New(conf.Google.ClientId, conf.Google.Secret, conf.Google.RedirectUri),
	)

	r := newRoom()
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)

	go r.run()

	//webサーバの開始
	log.Println("webサーバを開始します。ポート：", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServer:", err)
	}
}
