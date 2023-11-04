package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/cmdduplicate"
	"immich-go/cmdmetadata"
	"immich-go/cmdstack"
	"immich-go/cmdtool"
	"immich-go/cmdupload"
	"immich-go/immich"
	"immich-go/immich/logger"
	"os"
	"os/signal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Printf("immich-go  %s, commit %s, built at %s\n", version, commit, date)
	var err error
	var log = logger.NewLogger(logger.OK, true, false)
	// Create a context with cancel function to gracefully handle Ctrl+C events
	ctx, cancel := context.WithCancel(context.Background())

	// Handle Ctrl+C signal (SIGINT)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	go func() {
		<-signalChannel
		fmt.Println("\nCtrl+C received. Shutting down...")
		cancel() // Cancel the context when Ctrl+C is received
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		log, err = Run(ctx, log)
	}
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	log.OK("Done.")
}

type Application struct {
	Server      string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API         string // Immich api endpoint (http://container_ip:3301)
	Key         string // API Key
	DeviceUUID  string // Set a device UUID
	ApiTrace    bool   // Enable API call traces
	NoLogColors bool   // Disable log colors
	LogLevel    string // Idicate the log level
	Debug       bool   // Enable the debug mode

	Immich *immich.ImmichClient // Immich client
	Logger *logger.Logger       // Program's logger

}

func Run(ctx context.Context, log *logger.Logger) (*logger.Logger, error) {
	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return log, err
	}

	app := Application{}
	flag.StringVar(&app.Server, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	flag.StringVar(&app.API, "api", "", "Immich api endpoint (http://container_ip:3301)")
	flag.StringVar(&app.Key, "key", "", "API Key")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")
	flag.BoolVar(&app.NoLogColors, "no-colors-log", false, "Disable colors on logs")
	flag.StringVar(&app.LogLevel, "log-level", "ok", "Log level (Error|Warning|OK|Info), default OK")
	flag.BoolVar(&app.ApiTrace, "api-trace", false, "enable api call traces")
	flag.BoolVar(&app.Debug, "debug", false, "enable debug messages")
	flag.Parse()

	switch {
	case len(app.Server) == 0 && len(app.API) == 0:
		err = errors.Join(err, errors.New("missing -server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
	case len(app.Server) > 0 && len(app.API) > 0:
		err = errors.Join(err, errors.New("give either the -server or the -api option"))
	}
	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("missing -key"))
	}

	logLevel, e := logger.StringToLevel(app.LogLevel)
	if err != nil {
		err = errors.Join(err, e)
	}

	if len(flag.Args()) == 0 {
		err = errors.Join(err, errors.New("missing command upload|duplicate|stack"))
	}

	app.Logger = logger.NewLogger(logLevel, app.NoLogColors, app.Debug)

	if err != nil {
		return app.Logger, err
	}

	app.Immich, err = immich.NewImmichClient(app.Server, app.Key)
	if err != nil {
		return app.Logger, err
	}
	if app.API != "" {
		app.Immich.SetEndPoint(app.API)
	}
	if app.ApiTrace {
		app.Immich.EnableAppTrace(true)
	}

	err = app.Immich.PingServer(ctx)
	if err != nil {
		return app.Logger, err
	}
	app.Logger.OK("Server status: OK")

	user, err := app.Immich.ValidateConnection(ctx)
	if err != nil {
		return app.Logger, err
	}
	app.Logger.Info("Connected, user: %s", user.Email)

	cmd := flag.Args()[0]
	switch cmd {
	case "upload":
		err = cmdupload.UploadCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "duplicate":
		err = cmdduplicate.DuplicateCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "metadata":
		err = cmdmetadata.MetadataCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "stack":
		err = cmdstack.NewStackCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "tool":
		err = cmdtool.CommandTool(ctx, app.Immich, app.Logger, flag.Args()[1:])
	default:
		err = fmt.Errorf("unknwon command: %q", cmd)
	}
	return app.Logger, err
}
