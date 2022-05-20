package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/browser"
	"github.com/spf13/viper"
)

const serverURL = "https://github.com/login"

type Token struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

func authURL() string {
	return serverURL + "/oauth/authorize"
}

func tokenURL() string {
	return serverURL + "/oauth/access_token"
}

func getToken(code string) (*Token, error) {
	httpClient := http.Client{}

	params := url.Values{}
	params.Add("client_id", viper.GetString("client_id"))
	params.Add("client_secret", viper.GetString("client_secret"))
	params.Add("code", code)
	reqURL := fmt.Sprintf("%s?%s", tokenURL(), params.Encode())

	req, err := http.NewRequest(http.MethodPost, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	token := Token{}
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func authHandler(cancel context.CancelFunc, callback func(context.CancelFunc, *Token)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		code := r.FormValue("code")

		token, err := getToken(code)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><script>window.close();</script></html>")

		callback(cancel, token)
	}
}

func callback(cancel context.CancelFunc, token *Token) {
	cancel()
	log.Printf("%+v\n", token)
}

func main() {
	viper.AddConfigPath("./")
	viper.SetConfigName("github.conf")
	viper.SetConfigType("toml")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Fatal error config file: %s \n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := http.Server{Addr: "localhost:8080"}
	r := http.NewServeMux()
	r.HandleFunc("/oauth/redirect", authHandler(cancel, callback))
	s.Handler = r

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	params := url.Values{}
	params.Add("client_id", viper.GetString("client_id"))
	params.Add("redirect_uri", "http://localhost:8080/oauth/redirect")

	loginURL := fmt.Sprintf("%s?%s", authURL(), params.Encode())
	log.Println(loginURL)
	browser.OpenURL(loginURL)

	select {
	case <-ctx.Done():
		s.Shutdown(ctx)
	}
	log.Printf("done")
}
