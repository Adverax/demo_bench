package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"work/train/service"
)

func main() {
	conf, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background() // todo: tune ctx
	srv := service.New(service.NewMessenger(), conf)

	http.HandleFunc("/sites", actionSites(ctx, srv))
	http.ListenAndServe(":80", nil)

	// Поскольку в приведенном задании отсутствует ссылка на пакет goquery, то проект не может быть скомпилирован
}

func actionSites(
	ctx context.Context,
	srv service.Service,
) func(w http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, request *http.Request) {
		query := request.URL.Query().Get("search")
		m, err := srv.Execute(ctx, query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := json.Marshal(m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Charset", "utf-8")
		_, _ = w.Write(data)
	}
}

func loadConfig() (*service.Config, error) {
	var conf = &service.Config{
		Timeout: 30,
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(workDir, "conf.json"))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
