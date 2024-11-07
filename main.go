package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"goauthentik.io/api/v3"
	"log"
	"net/http"
	"strings"
)

type WebFingerClient struct {
	client      *api.APIClient
	Application string
}

func NewWebFingerClient(host string, token string, userAgent string, application string) WebFingerClient {
	cfg := api.NewConfiguration()
	//cfg.Debug = true
	cfg.Host = host
	cfg.Scheme = "https"
	cfg.UserAgent = userAgent
	cfg.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))

	client := api.NewAPIClient(cfg)

	return WebFingerClient{client: client, Application: application}
}

type WebFingerResource struct {
	Subject    string            `json:"subject"`
	Aliases    []string          `json:"aliases"`
	Properties map[string]string `json:"properties"`
	Links      []Link            `json:"links"`
}

type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func (wf *WebFingerClient) PokeAccount(account string) (*WebFingerResource, error) {
	resp, r, err := wf.client.CoreApi.CoreUsersList(context.Background()).Email(account).PageSize(1).IsActive(true).Execute()

	if err != nil {
		return nil, fmt.Errorf("error when calling `CoreApi.CoreUsersList`: %v. Full HTTP response: %v", err, r)
	}

	users, usersOk := resp.GetResultsOk()
	if !usersOk {
		return nil, errors.New("response did not contain any users")
	}

	if len(users) == 0 {
		return nil, errors.New("could not find account")
	}

	user := users[0]

	return &WebFingerResource{
		Subject: fmt.Sprintf("acct:%s", *user.Email),
		Aliases: []string{},
		Properties: map[string]string{
			"http://webfinger.example/ns/name": user.Name,
		},
		Links: []Link{
			{
				Rel:  "http://openid.net/specs/connect/1.0/issuer",
				Href: fmt.Sprintf("https://%s/application/o/%s/", wf.client.GetConfig().Host, wf.Application),
			},
			//{
			//	Rel:  "authorization_endpoint",
			//	Href: fmt.Sprintf("https://%s/application/o/%s/oauth2/authorize", wf.client.GetConfig().Host, wf.Application),
			//},
			//{
			//	Rel:  "token_endpoint",
			//	Href: fmt.Sprintf("https://%s/application/o/%s/oauth2/token", wf.client.GetConfig().Host, wf.Application),
			//},
			//{
			//	Rel: "userinfo_endpoint",
			//	Href: fmt.Sprintf("https://%s/application/o/%s/userinfo", wf.client.GetConfig().Host, wf.Application),
			//},
			//{
			//	Rel: "jwks_uri",
			//	Href: fmt.Sprintf("https://%s/application/o/%s/jwks", wf.client.GetConfig().Host, wf.Application),
			//},
			{
				Rel:  "http://webfinger.net/rel/avatar",
				Href: user.Avatar,
			},
		},
	}, nil
}

type config struct {
	Host                 string
	AuthentikHost        string
	Token                string
	UserAgent            string
	AuthentikApplication string
}

func readConfig() (*config, error) {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AUTHENTIK_WEBFINGER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	viper.SetDefault("Host", ":8080")

	viper.SetConfigName("config")
	//viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/authentik-webfinger")
	viper.AddConfigPath("$HOME/.authentik-webfinger")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Config file not found: %v\n", err)
		} else {
			log.Fatalf("Could not load config file: %v\n", err)
		}
		return nil, err
	}

	var cfg config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Could not unmarshal config: %v\n", err)
		return nil, err
	}

	//viper.OnConfigChange(func(e fsnotify.Event) {
	//	fmt.Println("Config file changed: ", e.Name)
	//})
	//viper.WatchConfig()

	return &cfg, nil
}

func main() {
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("Error when loading configuration: %v\n", err)
	}

	client := NewWebFingerClient(
		cfg.AuthentikHost,
		cfg.Token,
		cfg.UserAgent,
		cfg.AuthentikApplication,
	)

	http.HandleFunc("GET /.well-known/webfinger", func(writer http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()

		if !query.Has("resource") {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("resource not found in query"))
			return
		}

		resource := query.Get("resource")
		if !strings.HasPrefix(resource, "acct:") {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("resource does not have acct: prefix"))
			return
		}

		acct := strings.SplitAfter(resource, "acct:")[1]
		webfinger, err := client.PokeAccount(acct)

		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error when trying to fetch webfinger data: %v\n", err)
			return
		}

		bytes, err := json.Marshal(webfinger)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			log.Printf("Could not marshal webfinger: %v", err)
			return
		}

		writer.Header().Set("Content-Type", "application/jrd+json")
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.WriteHeader(http.StatusOK)
		writer.Write(bytes)
	})

	log.Fatal(http.ListenAndServe(cfg.Host, nil))
}
