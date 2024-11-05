package config

import (
	goflag "flag"
	"fmt"
	"github.com/lib/pq"
	// "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Dispatcher2Conf is the global conf
var Dispatcher2Conf Config
var SkipRequestProcessing *bool
var SkipScheduleProcessing *bool
var ServersConfigMap = make(map[string]ServerConf)

func init() {
	// ./go_dispatcher2 --config-file /etc/dispatcher2/dispatcher2.conf
	var configFilePath, configDir, conf_dDir string
	curentOS := runtime.GOOS
	switch curentOS {
	case "windows":
		configDir = "C:\\ProgramData\\Dispatcher2go"
		configFilePath = "C:\\ProgramData\\Dispatcher2go\\dispatcher2.yml"
		conf_dDir = "C:\\ProgramData\\Dispatcher2go\\conf.d"
	case "darwin", "linux":
		configFilePath = "/etc/dispatcher2go/dispatcher2.yml"
		configDir = "/etc/dispatcher2go/"
		conf_dDir = "/etc/dispatcher2go/conf.d" // for the conf.d directory where to dump server confs
	default:
		fmt.Println("Unsupported operating system")
		return
	}

	configFile := flag.String("config-file", configFilePath,
		"The path to the configuration file of the application")

	SkipRequestProcessing = flag.Bool("skip-request-processing", false, "Whether to skip requests processing")
	SkipScheduleProcessing = flag.Bool("skip-schedule-processing", false, "Whether to skip schedule processing")

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "9090")
	viper.SetDefault("server.dhis2_job_status_check_interval", 15)
	viper.SetDefault("server.request_process_interval", 5)
	viper.SetDefault("server.max_retries", 3)
	viper.SetDefault("server.retry_cron_expression", "*/5 * * * *")
	viper.SetDefault("server.timezone", "Africa/Kampala")

	viper.SetConfigName("dispatcher2")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	if len(*configFile) > 0 {
		viper.SetConfigFile(*configFile)
		log.Printf("Config File %v", *configFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// log.Fatalf("Configuration File: %v Not Found", *configFile, err)
			panic(fmt.Errorf("Fatal Error %w \n", err))

		} else {
			log.Fatalf("Error Reading Config: %v", err)

		}
	}

	err := viper.Unmarshal(&Dispatcher2Conf)
	if err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		err = viper.ReadInConfig()
		if err != nil {
			log.Fatalf("unable to reread configuration into global conf: %v", err)
		}
		_ = viper.Unmarshal(&Dispatcher2Conf)
		log.Printf(">>>>>+++++++++ %v", viper.GetInt("server.request_process_interval"))
	})
	viper.WatchConfig()

	v := viper.New()
	v.SetConfigType("json")

	fileList, err := getFilesInDirectory(conf_dDir)
	if err != nil {
		log.WithError(err).Info("Error reading directory")
	}

	// Loop through the files and read each one
	for _, file := range fileList {
		v.SetConfigFile(file)

		if err := v.ReadInConfig(); err != nil {
			log.WithError(err).WithField("File", file).Error("Error reading config file:")
			continue
		}

		// Unmarshal the config data into your structure
		var config ServerConf
		if err := v.Unmarshal(&config); err != nil {
			log.WithError(err).WithField("File", file).Error("Error unmarshaling config file:")
			continue
		}
		ServersConfigMap[config.Name] = config

		// Now you can use the config structure as needed
		// fmt.Printf("Configuration from %s: %+v\n", file, config)
	}
	v.OnConfigChange(func(e fsnotify.Event) {
		if err := v.ReadInConfig(); err != nil {
			log.WithError(err).WithField("File", e.Name).Error("Error reading config file:")
		}

		// Unmarshal the config data into your structure
		var config ServerConf
		if err := v.Unmarshal(&config); err != nil {
			log.WithError(err).WithField("File", e.Name).Fatalf("Error unmarshaling config file:")
		}
		ServersConfigMap[config.Name] = config
	})
	v.WatchConfig()
}

// Config is the top level cofiguration object
type Config struct {
	Database struct {
		URI        string `mapstructure:"uri" env:"DISPATCHER2_DB" env-default:"postgres://postgres:postgres@localhost/dispatcher2?sslmode=disable"`
		DBHost     string `mapstructure:"db_host" env:"DISPATCHER2_DB_HOST" env-default:"localhost"`
		DBPort     string `mapstructure:"db_port" env:"DISPATCHER2_DB_PORT" env-default:"5432"`
		DBUsername string `mapstructure:"db_username" env:"DISPATCHER2_USER" env-description:"API user name"`
		DBPassword string `mapstructure:"db_password" env:"DISPATCHER2_PASSWORD" env-description:"API user password"`
	} `yaml:"database"`

	Server struct {
		Host                        string `mapstructure:"host" env:"DISPATCHER2_HOST" env-default:"localhost"`
		Port                        string `mapstructure:"http_port" env:"DISPATCHER2_SERVER_PORT" env-description:"Server port" env-default:"9090"`
		ProxyPort                   string `mapstructure:"proxy_port" env:"DISPATCHER2_PROXY_PORT" env-description:"Server port" env-default:"9191"`
		MaxRetries                  int    `mapstructure:"max_retries" env:"DISPATCHER2_MAX_RETRIES" env-default:"3"`
		StartOfSubmissionPeriod     string `mapstructure:"start_submission_period" env:"START_SUBMISSION_PERIOD" env-default:"18"`
		EndOfSubmissionPeriod       string `mapstructure:"end_submission_period" env:"END_SUBMISSION_PERIOD" env-default:"24"`
		MaxConcurrent               int    `mapstructure:"max_concurrent" env:"DISPATCHER2_MAX_CONCURRENT" env-default:"5"`
		RetryCronExpression         string `mapstructure:"retry_cron_expression"  env:"RETRY_CRON_EXPRESSION" env-description:"The request retry Cron Expression" env-default:"*/5 * * * *"`
		RequestProcessInterval      int    `mapstructure:"request_process_interval" env:"REQUEST_PROCESS_INTERVAL" env-default:"4"`
		Dhis2JobStatusCheckInterval int    `mapstructure:"dhis2_job_status_check_interval" env:"DHIS2_JOB_STATUS_CHECK_INTERVAL" env-description:"The DHIS2 job status check interval in seconds" env-default:"30"`
		LogDirectory                string `mapstructure:"logdir" env:"DISPATCHER2_LOGDIR" env-default:"/var/log/dispatcher2"`
		UseSSL                      string `mapstructure:"use_ssl" env:"DISPATCHER2_USE_SSL" env-default:""`
		SSLClientCertKeyFile        string `mapstructure:"ssl_client_certkey_file" env:"SSL_CLIENT_CERTKEY_FILE" env-default:""`
		SSLServerCertKeyFile        string `mapstructure:"ssl_server_certkey_file" env:"SSL_SERVER_CERTKEY_FILE" env-default:""`
		SSLTrustedCAFile            string `mapstructure:"ssl_trusted_cafile" env:"SSL_TRUSTED_CA_FILE" env-default:""`
		TimeZone                    string `mapstructure:"timezone" env:"DISPATCHER2_TIMEZONE" env-default:"Africa/Kampala" env-description:"The time zone used for this dispatcher2 deployment"`
	} `yaml:"server"`

	API struct {
		Email     string `yaml:"email" env:"DISPATCHER2_EMAIL" env-description:"API user email address"`
		AuthToken string `yaml:"authtoken" env:"RAPIDPRO_AUTH_TOKEN" env-description:"API JWT authorization token"`
		SmsURL    string `yaml:"smsurl" env:"SMS_URL" env-description:"API SMS endpoint"`
	} `yaml:"api"`
}

type ServerConf struct {
	ID                      int64          `mapstructure:"id" json:"-"`
	UID                     string         `mapstructure:"uid" json:"uid,omitempty"`
	Name                    string         `mapstructure:"name" json:"name" validate:"required"`
	Username                string         `mapstructure:"username" json:"username"`
	Password                string         `mapstructure:"password" json:"password,omitempty"`
	IsProxyServer           bool           `mapstructure:"isProxyserver" json:"isProxyServer,omitempty"`
	SystemType              string         `mapstructure:"systemType" json:"systemType,omitempty"`
	EndPointType            string         `mapstructure:"endpointType" json:"endPointType,omitempty"`
	AuthToken               string         `mapstructure:"authToken" db:"auth_token" json:"AuthToken"`
	IPAddress               string         `mapstructure:"IPAddress"  json:"IPAddress"`
	URL                     string         `mapstructure:"URL" json:"URL" validate:"required,url"`
	CCURLS                  pq.StringArray `mapstructure:"CCURLS" json:"CCURLS,omitempty"`
	CallbackURL             string         `mapstructure:"callbackURL" json:"callbackURL,omitempty"`
	HTTPMethod              string         `mapstructure:"HTTPMethod" json:"HTTPMethod" validate:"required"`
	AuthMethod              string         `mapstructure:"AuthMethod" json:"AuthMethod" validate:"required"`
	AllowCallbacks          bool           `mapstructure:"allowCallbacks" json:"allowCallbacks,omitempty"`
	AllowCopies             bool           `mapstructure:"allowCopies" json:"allowCopies,omitempty"`
	UseAsync                bool           `mapstructure:"useAsync" json:"useAsync,omitempty"`
	UseSSL                  bool           `mapstructure:"useSSL" json:"useSSL,omitempty"`
	ParseResponses          bool           `mapstructure:"parseResponses" json:"parseResponses,omitempty"`
	SSLClientCertKeyFile    string         `mapstructure:"sslClientCertkeyFile" json:"sslClientCertkeyFile"`
	StartOfSubmissionPeriod int            `mapstructure:"startSubmissionPeriod" json:"startSubmissionPeriod"`
	EndOfSubmissionPeriod   int            `mapstructure:"endSubmissionPeriod" json:"endSubmissionPeriod"`
	XMLResponseXPATH        string         `mapstructure:"XMLResponseXPATH"  json:"XMLResponseXPATH"`
	JSONResponseXPATH       string         `mapstructure:"JSONResponseXPATH" json:"JSONResponseXPATH"`
	Suspended               bool           `mapstructure:"suspended" json:"suspended,omitempty"`
	URLParams               map[string]any `mapstructure:"URLParams" json:"URLParams,omitempty"`
	Created                 time.Time      `mapstructure:"created" json:"created,omitempty"`
	Updated                 time.Time      `mapstructure:"updated" json:"updated,omitempty"`
	AllowedSources          []string       `mapstructure:"allowedSources" json:"allowedSources,omitempty"`
}

func getFilesInDirectory(directory string) ([]string, error) {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
