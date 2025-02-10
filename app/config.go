package chatter

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	"github.com/spf13/viper"
)

type Config struct {
	// Port is the Port number to listen on. The default is 8080.
	Port int `validate:"required,port" default:"8080"`
	// Hostname is the Hostname to listen on. The default is 0.0.0.0.
	Hostname string `validate:"required" default:"0.0.0.0"`
	Auth     struct {
		// Secret is the Secret key used to sign JWT tokens.
		// The secret must be a base64 encoded string. The default is a random 32 byte string.
		Secret Base64Encoded `validate:"required"`
	}
	SQLite struct {
		// File is the path to the SQLite database file.
		File string `validate:"required" `
		// Migrations is the path to the directory that the migration files reside.
		Migrations string `validate:"required" `
	}
	// AllowedOrigins is a list of origins that are allowed to connect to the server.
	// The default is ["*"].
	AllowedOrigins []string
	valid          bool
}

type Base64Encoded []byte

func (b *Base64Encoded) UnmarshalText(text []byte) error {
	dec, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}
	*b = dec
	return nil
}

// LoadConfig loads the configuration from the config file and environment variables.
// Any invalid configuration will not be loaded, and the error wil be cought in the validation step.
func LoadConfig() (*Config, error) {
	config := &Config{}
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("port", 8080)
	// generate a random secret key
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	viper.SetDefault("auth.secret", base64.StdEncoding.EncodeToString(secret))
	viper.SetDefault("hostname", "0.0.0.0")

	viper.SetDefault("sqlite.file", "./chatter.db")
	viper.SetDefault("sqlite.migrations", "./migrations")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := viper.Unmarshal(&config,
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
			mapstructure.TextUnmarshallerHookFunc(),
			mapstructure.StringToSliceHookFunc(",")),
		),
	); err != nil {
		// defer error to validation step
		return config, nil
	}
	return config, nil
}

func (c *Config) Validate() error {
	if c.valid {
		return nil
	}
	err := validate.Struct(c)
	if err != nil {
		return err
	}
	c.valid = true
	return nil
}

func FormatValidationErrors(err error) string {

	errors, ok := err.(validator.ValidationErrors)
	if !ok {
		return ""
	}
	trans, _ := uniTrans.GetTranslator("en")
	translated := errors.Translate(trans)

	var sb strings.Builder
	for v := range maps.Values(translated) {
		sb.WriteString(v)
		sb.WriteString("\n")
	}
	return sb.String()
}
