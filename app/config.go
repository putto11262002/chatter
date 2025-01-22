package chatter

import "reflect"

type Config struct {
	// Port is the Port number to listen on. The default is 8080.
	Port int
	// Hostname is the Hostname to listen on. The default is 0.0.0.0.
	Hostname string
	// Secret is the Secret key used to sign JWT tokens.
	// This must be a base64 encoded string.
	Secret []byte
	// SQLiteFile is the path to the SQLite database file.
	SQLiteFile string
	// MigrationDir is the path to the directory that the migration files reside.
	MigrationDir string
	// AllowedOrigins is a list of origins that are allowed to connect to the server.
	AllowedOrigins []string
}

func (c *Config) Validate() error {
	return validate.Struct(c)
}

func MergeConfig(c *Config, o *Config) *Config {
	c1V := reflect.ValueOf(c).Elem() // Dereference the pointer to access the struct
	c2V := reflect.ValueOf(o).Elem()

	for i := 0; i < c1V.NumField(); i++ {
		c1Field := c1V.Field(i)
		c2Field := c2V.Field(i)

		// Ensure the field is settable and c2 field is not zero
		if c1Field.CanSet() && !c2Field.IsZero() {
			c1Field.Set(c2Field)
		}
	}
	return c
}
