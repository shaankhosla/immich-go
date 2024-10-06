package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/tzone"
	"github.com/spf13/cobra"
)

type Client struct {
	Server        string        // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API           string        // Immich api endpoint (http://container_ip:3301)
	APIKey        string        // API Key
	APITrace      bool          // Enable API call traces
	SkipSSL       bool          // Skip SSL Verification
	ClientTimeout time.Duration // Set the client request timeout
	DeviceUUID    string        // Set a device UUID
	DryRun        bool          // Protect the server from changes
	TimeZone      string        // Override default TZ

	APITraceWriter     io.WriteCloser         // API tracer
	APITraceWriterName string                 // API trace log name
	Immich             immich.ImmichInterface // Immich client

	// NoUI               bool           // Disable user interface
	// DebugFileList      bool           // When true, the file argument is a file wile the list of Takeout files
}

// add server flags to the command cmd
func AddClientFlags(ctx context.Context, cmd *cobra.Command, app *Application) {
	client := app.Client()
	client.DeviceUUID, _ = os.Hostname()

	cmd.PersistentFlags().StringVarP(&client.Server, "server", "s", client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.PersistentFlags().StringVar(&client.API, "api", "", "Immich api endpoint (example http://container_ip:3301)")
	cmd.PersistentFlags().StringVarP(&client.APIKey, "api-key", "k", "", "API Key")
	cmd.PersistentFlags().BoolVar(&client.APITrace, "api-trace", false, "Enable trace of api calls")
	cmd.PersistentFlags().BoolVar(&client.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	cmd.PersistentFlags().DurationVar(&client.ClientTimeout, "client-timeout", 5*time.Minute, "Set server calls timeout")
	cmd.PersistentFlags().StringVar(&client.DeviceUUID, "device-uuid", client.DeviceUUID, "Set a device UUID")
	cmd.PersistentFlags().BoolVar(&client.DryRun, "dry-run", false, "Simulate all actions")
	cmd.PersistentFlags().StringVar(&client.TimeZone, "time-zone", client.TimeZone, "Override the system time zone")

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, StartClient, ctx, cmd, app)
}

func StartClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	client := app.Client()
	log := app.Log()

	var joinedErr error
	if client.Server != "" {
		client.Server = strings.TrimSuffix(client.Server, "/")
	}
	if client.TimeZone != "" {
		_, err := tzone.SetLocal(client.TimeZone)
		joinedErr = errors.Join(joinedErr, err)
	}

	// Plug the journal on the Log
	if log.File != "" {
		if log.writer == nil {
			err := configuration.MakeDirForFile(log.File)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			err = log.sLevel.UnmarshalText([]byte(strings.ToUpper(log.Level)))
			if err != nil {
				return err
			}
			log.SetLogWriter(f)
			log.writer = f
		}
	}

	// If the client isn't yet initialized
	if client.Immich == nil {
		switch {
		case client.Server == "" && client.API == "":
			joinedErr = errors.Join(joinedErr, errors.New("missing --server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
		case client.Server != "" && client.API != "":
			joinedErr = errors.Join(joinedErr, errors.New("give either the --server or the --api option"))
		}
		if client.APIKey == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing --API-key"))
		}

		if joinedErr != nil {
			return joinedErr
		}

		log.Info("Connection to the server " + client.Server)

		var err error
		client.Immich, err = immich.NewImmichClient(client.Server, client.APIKey, immich.OptionVerifySSL(client.SkipSSL), immich.OptionConnectionTimeout(client.ClientTimeout))
		if err != nil {
			return err
		}
		if client.API != "" {
			client.Immich.SetEndPoint(client.API)
		}
		if client.DeviceUUID != "" {
			client.Immich.SetDeviceUUID(client.DeviceUUID)
		}

		if client.APITrace {
			if client.APITraceWriter == nil {
				err := configuration.MakeDirForFile(log.File)
				if err != nil {
					return err
				}
				client.APITraceWriterName = strings.TrimSuffix(log.File, filepath.Ext(log.File)) + ".trace.log"
				client.APITraceWriter, err = os.OpenFile(client.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
				if err != nil {
					return err
				}
				client.Immich.EnableAppTrace(client.APITraceWriter)
			}
		}

		err = client.Immich.PingServer(ctx)
		if err != nil {
			return err
		}
		log.Info("Server status: OK")

		user, err := client.Immich.ValidateConnection(ctx)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Connected, user: %s", user.Email))
	}

	return nil
}
