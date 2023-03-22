package jwtauth

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.infratographer.com/x/viperx"
)

// MustViperFlags adds jwks-uri to the provided flagset and binds to viper jwks.uri.
func MustViperFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("jwks-uri", "", "URI to jwks json configuration.")
	viperx.MustBindFlag(v, "jwks.uri", flags.Lookup("jwks-uri"))
}
