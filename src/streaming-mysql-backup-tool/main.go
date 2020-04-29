package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"streaming-mysql-backup-tool/api"
	"streaming-mysql-backup-tool/collector"
	c "streaming-mysql-backup-tool/config"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/api/middleware"
)

func main() {
	config, err := c.NewConfig(os.Args)
	logger := config.Logger

	if err != nil {
		logger.Fatal("Failed to read config", err, lager.Data{
			"config": config,
		})
	}

	mux := http.NewServeMux()

	wrappedMux := middleware.Chain{
		middleware.NewBasicAuth(config.Credentials.Username, config.Credentials.Password),
	}.Wrap(mux)

	backupHandler := &api.BackupHandler{
		CommandGenerator:   config.Cmd,
		Logger:             logger,
		CollectorGenerator: func() api.Collector { return collector.NewCollector(collector.NewScriptExecutor(), logger) },
	}

	db, err := sql.Open("mysql", config.DB.DSN())
	if err != nil {
		logger.Fatal("Failed to initialize db: %s", err)
	}

	mux.Handle("/backup", backupHandler)

	mux.Handle("/metadata", api.Metadata(db))

	pidfile, err := os.Create(config.PidFile)
	if err != nil {
		logger.Fatal("Failed to create a file", err)
	}

	_ = ioutil.WriteFile(pidfile.Name(), []byte(strconv.Itoa(os.Getpid())), 0644)

	logger.Info("Starting server with configuration", lager.Data{
		"port": config.Port,
	})

	httpServer := &http.Server{
		Addr:      fmt.Sprintf(":%d", config.Port),
		Handler:   wrappedMux,
		TLSConfig: config.TLS.Config,
	}
	err = httpServer.ListenAndServeTLS("", "")
	logger.Fatal("Streaming backup tool has exited with an error", err)
}
